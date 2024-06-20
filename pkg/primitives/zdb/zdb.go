package zdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/zdb"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	nwmod "github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	// https://hub.grid.tf/api/flist/tf-autobuilder/threefoldtech-0-db-development.flist/light
	// To get the latest symlink pointer
	zdbFlistURL         = "https://hub.grid.tf/tf-autobuilder/threefoldtech-0-db-release-v2.0.8-55737c9202.flist"
	zdbContainerNS      = "zdb"
	zdbContainerDataMnt = "/zdb"
	zdbPort             = 9900
)

var uuidRegex = regexp.MustCompile(`([0-9a-f]{8})\b-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-\b[0-9a-f]{12}`)

// ZDB types
type ZDB = zos.ZDB

type tZDBContainer pkg.Container

type safeError struct {
	error
}

func newSafeError(err error) error {
	if err == nil {
		return nil
	}
	return &safeError{err}
}

func (se *safeError) Unwrap() error {
	return se.error
}

func (se *safeError) Error() string {
	return uuidRegex.ReplaceAllString(se.error.Error(), `$1-***`)
}

func (z *tZDBContainer) DataMount() (string, error) {
	for _, mnt := range z.Mounts {
		if mnt.Target == zdbContainerDataMnt {
			return mnt.Source, nil
		}
	}

	return "", fmt.Errorf("container '%s' does not have a valid data mount", z.Name)
}

var (
	_ provision.Manager     = (*Manager)(nil)
	_ provision.Updater     = (*Manager)(nil)
	_ provision.Initializer = (*Manager)(nil)
	_ provision.Pauser      = (*Manager)(nil)
)

type Manager struct {
	zbus zbus.Client
}

func NewManager(zbus zbus.Client) *Manager {
	return &Manager{zbus}
}

func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	res, err := p.zdbProvisionImpl(ctx, wl)
	return res, newSafeError(err)
}

func (p *Manager) zdbListContainers(ctx context.Context) (map[pkg.ContainerID]tZDBContainer, error) {
	contmod := stubs.NewContainerModuleStub(p.zbus)

	containerIDs, err := contmod.List(ctx, zdbContainerNS)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list running containers")
	}

	// for each container we try to find a free space to jam in this new zdb namespace
	// request
	m := make(map[pkg.ContainerID]tZDBContainer)

	for _, containerID := range containerIDs {
		container, err := contmod.Inspect(ctx, zdbContainerNS, containerID)
		if err != nil {
			log.Error().Err(err).Str("container-id", string(containerID)).Msg("failed to inspect zdb container")
			continue
		}
		cont := tZDBContainer(container)

		if _, err = cont.DataMount(); err != nil {
			log.Error().Err(err).Msg("failed to get data directory of zdb container")
			continue
		}
		m[containerID] = cont
	}

	return m, nil
}

func (p *Manager) zdbProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (zos.ZDBResult, error) {
	var (
		// contmod = stubs.NewContainerModuleStub(p.zbus)
		storage = stubs.NewStorageModuleStub(p.zbus)
		nsID    = wl.ID.String()
		config  ZDB
	)
	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to decode reservation schema")
	}

	// for each container we try to find a free space to jam in this new zdb namespace
	// request
	containers, err := p.zdbListContainers(ctx)
	if err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to list container data volumes")
	}

	var candidates []tZDBContainer
	// check if namespace already exist
	for id, container := range containers {
		dataPath, _ := container.DataMount() // the error should not happen

		index := zdb.NewIndex(dataPath)
		nss, err := index.Namespaces()
		if err != nil {
			// skip or error
			log.Error().Err(err).Str("container-id", string(id)).Msg("couldn't list namespaces")
			continue
		}

		for _, ns := range nss {
			if ns.Name != nsID {
				continue
			}

			containerIPs, err := p.waitZDBIPs(ctx, container.Network.Namespace, container.CreatedAt)
			if err != nil {
				return zos.ZDBResult{}, errors.Wrap(err, "failed to find IP address on zdb0 interface")
			}

			return zos.ZDBResult{
				Namespace: nsID,
				IPs:       ipsToString(containerIPs),
				Port:      zdbPort,
			}, nil
		}

		device, err := storage.DeviceLookup(ctx, container.Name)
		if err != nil {
			log.Error().Err(err).Str("container", string(id)).Msg("failed to inspect zdb device")
			continue
		}

		if uint64(device.Usage.Used)+uint64(config.Size) <= uint64(device.Usage.Size) {
			candidates = append(candidates, container)
		}
	}

	var cont tZDBContainer
	if len(candidates) > 0 {
		cont = candidates[0]
	} else {
		// allocate new disk
		device, err := storage.DeviceAllocate(ctx, config.Size)
		if err != nil {
			return zos.ZDBResult{}, errors.Wrap(err, "couldn't allocate device to satisfy namespace size")
		}
		cont, err = p.ensureZdbContainer(ctx, device)
		if err != nil {
			return zos.ZDBResult{}, errors.Wrap(err, "failed to start zdb container")
		}
	}

	containerIPs, err := p.waitZDBIPs(ctx, cont.Network.Namespace, cont.CreatedAt)
	if err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to find IP address on zdb0 interface")
	}

	log.Warn().Msgf("ip for zdb containers %s", containerIPs)
	// this call will actually configure the namespace in zdb and set the password
	if err := p.createZDBNamespace(pkg.ContainerID(cont.Name), nsID, config); err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to create zdb namespace")
	}

	return zos.ZDBResult{
		Namespace: nsID,
		IPs:       ipsToString(containerIPs),
		Port:      zdbPort,
	}, nil
}

func (p *Manager) ensureZdbContainer(ctx context.Context, device pkg.Device) (tZDBContainer, error) {
	container := stubs.NewContainerModuleStub(p.zbus)
	name := pkg.ContainerID(device.ID)

	cont, err := container.Inspect(ctx, zdbContainerNS, name)
	if err != nil && strings.Contains(err.Error(), "not found") {
		// container not found, create one
		if err := p.createZdbContainer(ctx, device); err != nil {
			return tZDBContainer(cont), err
		}
		cont, err = container.Inspect(ctx, zdbContainerNS, name)
		if err != nil {
			return tZDBContainer{}, err
		}
	} else if err != nil {
		// other error
		return tZDBContainer{}, err
	}

	return tZDBContainer(cont), nil
}

func (p *Manager) zdbRootFS(ctx context.Context) (string, error) {
	flist := stubs.NewFlisterStub(p.zbus)
	var err error
	var rootFS string

	hash, err := flist.FlistHash(ctx, zdbFlistURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to get flist hash")
	}

	rootFS, err = flist.Mount(ctx, hash, zdbFlistURL, pkg.MountOptions{
		Limit:    10 * gridtypes.Megabyte,
		ReadOnly: false,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to mount zdb flist")
	}

	return rootFS, nil
}

func (p *Manager) createZdbContainer(ctx context.Context, device pkg.Device) error {
	var (
		name       = pkg.ContainerID(device.ID)
		cont       = stubs.NewContainerModuleStub(p.zbus)
		flist      = stubs.NewFlisterStub(p.zbus)
		volumePath = device.Path
		network    = stubs.NewNetworkerStub(p.zbus)

		slog = log.With().Str("containerID", string(name)).Logger()
	)

	slog.Debug().Str("flist", zdbFlistURL).Msg("mounting flist")

	rootFS, err := p.zdbRootFS(ctx)
	if err != nil {
		return err
	}

	cleanup := func() {
		if err := cont.Delete(ctx, zdbContainerNS, name); err != nil {
			slog.Error().Err(err).Msg("failed to delete 0-db container")
		}

		if err := flist.Unmount(ctx, string(name)); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}
	}

	// create the network namespace and macvlan for the 0-db container
	netNsName, err := network.EnsureZDBPrepare(ctx, device.ID)
	if err != nil {
		if err := flist.Unmount(ctx, string(name)); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}

		return errors.Wrap(err, "failed to prepare zdb network")
	}

	socketDir := socketDir(name)
	if err := os.MkdirAll(socketDir, 0550); err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "failed to create directory: %s", socketDir)
	}

	cl := zdbConnection(name)
	if err := cl.Connect(); err == nil {
		// it seems there is a running container already
		cl.Close()
		return nil
	}

	// make sure the file does not exist otherwise we get the address already in use error
	if err := os.Remove(socketFile(name)); err != nil && !os.IsNotExist(err) {
		return err
	}

	cmd := fmt.Sprintf("/bin/zdb --protect --admin '%s' --data /zdb/data --index /zdb/index  --listen :: --port %d --socket /socket/zdb.sock --dualnet", device.ID, zdbPort)

	err = p.zdbRun(ctx, string(name), rootFS, cmd, netNsName, volumePath, socketDir)
	if err != nil {
		cleanup()
		return errors.Wrap(err, "failed to create container")
	}

	cl = zdbConnection(name)
	defer cl.Close()

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second * 20
	bo.MaxElapsedTime = time.Minute * 2

	if err := backoff.RetryNotify(cl.Connect, bo, func(err error, d time.Duration) {
		log.Debug().Err(err).Str("duration", d.String()).Msg("waiting for zdb to start")
	}); err != nil {
		cleanup()
		return errors.Wrapf(err, "failed to establish connection to zdb")
	}

	return nil
}

// dataMigration will make sure that we delete any data files from v1. This is
// hardly a data migration but at this early stage it's fine since there is still
// no real data loads live on the grid. All v2 zdbs, will be safe.
func (p *Manager) dataMigration(ctx context.Context, volume string) {
	v1, _ := zdb.IsZDBVersion1(ctx, volume)
	// TODO: what if there is an error?
	if !v1 {
		// it's eather a new volume, or already on version 2
		// so nothing to do.
		return
	}

	for _, sub := range []string{"data", "index"} {
		if err := os.RemoveAll(filepath.Join(volume, sub)); err != nil {
			log.Error().Err(err).Msg("failed to delete obsolete data directories")
		}
	}
}

func (p *Manager) zdbRun(ctx context.Context, name string, rootfs string, cmd string, netns string, volumepath string, socketdir string) error {
	cont := stubs.NewContainerModuleStub(p.zbus)

	// we do data migration here because this is called
	// on new zdb starts, or updating the runtime.
	p.dataMigration(ctx, volumepath)

	conf := pkg.Container{
		Name:        name,
		RootFS:      rootfs,
		Entrypoint:  cmd,
		Interactive: false,
		Network:     pkg.NetworkInfo{Namespace: netns},
		Mounts: []pkg.MountInfo{
			{
				Source: volumepath,
				Target: zdbContainerDataMnt,
			},
			{
				Source: socketdir,
				Target: "/socket",
			},
		},
	}

	_, err := cont.Run(
		ctx,
		zdbContainerNS,
		conf,
	)

	return err
}

func (p *Manager) waitZDBIPs(ctx context.Context, namespace string, created time.Time) ([]net.IP, error) {
	// TODO: this method need to be abstracted, since it's now depends on the knewledge
	// of the networking daemon internal (interfaces names)
	// may be at least just get all ips from all interfaces inside the namespace
	// will be a slightly better solution
	var (
		network      = stubs.NewNetworkerStub(p.zbus)
		containerIPs []net.IP
	)

	log.Debug().Time("created-at", created).Str("namespace", namespace).Msg("checking zdb container ips")
	getIP := func() error {
		// some older setups that might still be running has PubIface set to zdb0 not eth0
		// so we need to make sure that this we also try this older name
		ips, _, err := network.Addrs(ctx, nwmod.PubIface, namespace)
		if err != nil {
			var err2 error
			ips, _, err2 = network.Addrs(ctx, "zdb0", namespace)
			if err2 != nil {
				log.Debug().Err(err).Msg("no public ip found, waiting")
				return err
			}
		}

		yggIps, _, err := network.Addrs(ctx, nwmod.ZDBYggIface, namespace)
		if err != nil {
			return err
		}
		ips = append(ips, yggIps...)

		MyceliumIps, _, err := network.Addrs(ctx, nwmod.ZDBMyceliumIface, namespace)
		if err != nil {
			return err
		}
		ips = append(ips, MyceliumIps...)

		var (
			public   = false
			ygg      = false
			mycelium = false
		)
		containerIPs = containerIPs[:0]

		for _, ip := range ips {
			if isPublic(ip) && !isYgg(ip) && !isMycelium(ip) {
				log.Warn().IPAddr("ip", ip).Msg("0-db container public ip found")
				public = true
				containerIPs = append(containerIPs, ip)
			}
			if isYgg(ip) {
				log.Warn().IPAddr("ip", ip).Msg("0-db container ygg ip found")
				ygg = true
				containerIPs = append(containerIPs, ip)
			}
			if isMycelium(ip) {
				log.Warn().IPAddr("ip", ip).Msg("0-db container mycelium ip found")
				mycelium = true
				containerIPs = append(containerIPs, ip)
			}
		}

		log.Warn().Msgf("public %v ygg: %v mycelium: %v", public, ygg, mycelium)
		if public && ygg && mycelium || time.Since(created) > 2*time.Minute {
			// if we have all ips detected or if the container is older than 2 minutes
			// so it's safe we assume ips are final
			return nil
		}
		return fmt.Errorf("waiting for more addresses")
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Minute
	bo.MaxElapsedTime = time.Minute * 2

	if err := backoff.RetryNotify(getIP, bo, func(err error, d time.Duration) {
		log.Debug().Err(err).Str("duration", d.String()).Msg("failed to get zdb public IP")
	}); err != nil && len(containerIPs) == 0 {
		return nil, errors.Wrapf(err, "failed to get an IP for interface")
	}

	return containerIPs, nil
}

func (p *Manager) createZDBNamespace(containerID pkg.ContainerID, nsID string, config ZDB) error {
	zdbCl := zdbConnection(containerID)
	defer zdbCl.Close()
	if err := zdbCl.Connect(); err != nil {
		return errors.Wrapf(err, "failed to connect to 0-db: %s", containerID)
	}

	exists, err := zdbCl.Exist(nsID)
	if err != nil {
		return err
	}
	if !exists {
		if err := zdbCl.CreateNamespace(nsID); err != nil {
			return errors.Wrapf(err, "failed to create namespace in 0-db: %s", containerID)
		}

		if err := zdbCl.NamespaceSetMode(nsID, string(config.Mode)); err != nil {
			return errors.Wrap(err, "failed to set namespace mode")
		}
	}

	if config.Password != "" {
		if err := zdbCl.NamespaceSetPassword(nsID, config.Password); err != nil {
			return errors.Wrapf(err, "failed to set password namespace %s in 0-db: %s", nsID, containerID)
		}
	}

	if err := zdbCl.NamespaceSetPublic(nsID, config.Public); err != nil {
		return errors.Wrapf(err, "failed to make namespace %s public in 0-db: %s", nsID, containerID)
	}

	if err := zdbCl.NamespaceSetSize(nsID, uint64(config.Size)); err != nil {
		return errors.Wrapf(err, "failed to set size on namespace %s in 0-db: %s", nsID, containerID)
	}

	return nil
}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	return newSafeError(p.zdbDecommissionImpl(ctx, wl))
}

func (p *Manager) zdbDecommissionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	containers, err := p.zdbListContainers(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to list running zdbs")
	}

	for id, container := range containers {
		con := zdbConnection(id)
		if err := con.Connect(); err == nil {
			defer con.Close()
			if ok, _ := con.Exist(wl.ID.String()); ok {
				if err := con.DeleteNamespace(wl.ID.String()); err != nil {
					return errors.Wrap(err, "failed to delete namespace")
				}
			}

			continue
		}
		// if we failed to connect, may be check the data directory if the namespace exists
		data, err := container.DataMount()
		if err != nil {
			log.Error().Err(err).Str("container-id", string(id)).Msg("failed to get container data directory")
			return err
		}

		idx := zdb.NewIndex(data)
		if !idx.Exists(wl.ID.String()) {
			continue
		}

		return idx.Delete(wl.ID.String())
	}

	return nil
}

func (p *Manager) findContainer(ctx context.Context, name string) (zdb.Client, error) {
	containers, err := p.zdbListContainers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list running zdbs")
	}

	for id := range containers {
		cl := zdbConnection(id)
		if err := cl.Connect(); err != nil {
			log.Error().Err(err).Str("id", string(id)).Msg("failed to connect to zdb instance")
			continue
		}

		if ok, _ := cl.Exist(name); ok {
			return cl, nil
		}

		_ = cl.Close()
	}

	return nil, os.ErrNotExist
}

func (p *Manager) Pause(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	cl, err := p.findContainer(ctx, wl.ID.String())
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return provision.UnChanged(err)
	}

	defer cl.Close()

	if err := cl.NamespaceSetLock(wl.ID.String(), true); err != nil {
		return provision.UnChanged(errors.Wrap(err, "failed to set namespace locking"))
	}

	return provision.Paused()
}

func (p *Manager) Resume(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	cl, err := p.findContainer(ctx, wl.ID.String())
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return provision.UnChanged(err)
	}

	defer cl.Close()

	if err := cl.NamespaceSetLock(wl.ID.String(), false); err != nil {
		return provision.UnChanged(errors.Wrap(err, "failed to set namespace locking"))
	}

	return provision.Ok()
}

func (p *Manager) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	res, err := p.zdbUpdateImpl(ctx, wl)
	return res, newSafeError(err)
}

func (p *Manager) zdbUpdateImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.ZDBResult, err error) {
	current, err := provision.GetWorkload(ctx, wl.Name)
	if err != nil {
		// this should not happen but we need to have the check anyway
		return result, errors.Wrapf(err, "no zdb workload with name '%s' is deployed", wl.Name.String())
	}

	var old ZDB
	if err := json.Unmarshal(current.Data, &old); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	var new ZDB
	if err := json.Unmarshal(wl.Data, &new); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	if new.Mode != old.Mode {
		return result, provision.UnChanged(fmt.Errorf("cannot change namespace mode"))
	}

	if new.Size < old.Size {
		// technically shrinking a namespace is possible, but the problem is if you set it
		// to a size smaller than actual size used by the namespace, zdb will just not accept
		// writes anymore but NOT change the effective size.
		// While this makes sense fro ZDB. it will not make zos able to calculate the namespace
		// consumption because it will only count the size used by the workload from the user
		// data but not actual size on disk. hence shrinking is not allowed.
		return result, provision.UnChanged(fmt.Errorf("cannot shrink zdb namespace"))
	}

	if new.Size == old.Size && new.Password == old.Password && new.Public == old.Public {
		// unnecessary update.
		return result, provision.ErrNoActionNeeded
	}
	containers, err := p.zdbListContainers(ctx)
	if err != nil {
		return result, provision.UnChanged(errors.Wrap(err, "failed to list running zdbs"))
	}

	name := wl.ID.String()
	for id, container := range containers {
		con := zdbConnection(id)
		if err := con.Connect(); err != nil {
			log.Error().Err(err).Str("id", string(id)).Msg("failed t connect to zdb")
			continue
		}
		defer con.Close()
		if ok, _ := con.Exist(name); !ok {
			continue
		}

		// we try the size first because this can fail easy
		if new.Size != old.Size {
			free, reserved, err := reservedSpace(con)
			if err != nil {
				return result, provision.UnChanged(errors.Wrap(err, "failed to calculate free/reserved space from zdb"))
			}

			if reserved+new.Size-old.Size > free {
				return result, provision.UnChanged(fmt.Errorf("no enough free space to support new size"))
			}

			if err := con.NamespaceSetSize(name, uint64(new.Size)); err != nil {
				return result, provision.UnChanged(errors.Wrap(err, "failed to set new zdb namespace size"))
			}
		}

		// this is kinda proplamatic because what if we changed the size for example, but failed
		// to setup the password
		if new.Password != old.Password {
			if err := con.NamespaceSetPassword(name, new.Password); err != nil {
				return result, provision.UnChanged(errors.Wrap(err, "failed to set new password"))
			}
		}
		if new.Public != old.Public {
			if err := con.NamespaceSetPublic(name, new.Public); err != nil {
				return result, provision.UnChanged(errors.Wrap(err, "failed to set public flag"))
			}
		}

		containerIPs, err := p.waitZDBIPs(ctx, container.Network.Namespace, container.CreatedAt)
		if err != nil {
			return zos.ZDBResult{}, errors.Wrap(err, "failed to find IP address on zdb0 interface")
		}

		return zos.ZDBResult{
			Namespace: name,
			IPs:       ipsToString(containerIPs),
			Port:      zdbPort,
		}, nil

	}

	return result, fmt.Errorf("namespace not found")
}

func socketDir(containerID pkg.ContainerID) string {
	return fmt.Sprintf("/var/run/zdb_%s", containerID)
}

func socketFile(containerID pkg.ContainerID) string {
	return filepath.Join(socketDir(containerID), "zdb.sock")
}

// we declare this method as a variable so we can
// mock it in testing.
var zdbConnection = func(id pkg.ContainerID) zdb.Client {
	socket := fmt.Sprintf("unix://%s@%s", string(id), socketFile(id))
	return zdb.New(socket)
}

func reservedSpace(con zdb.Client) (free, reserved gridtypes.Unit, err error) {
	nss, err := con.Namespaces()
	if err != nil {
		return 0, 0, err
	}

	for _, id := range nss {
		ns, err := con.Namespace(id)
		if err != nil {
			return 0, 0, err
		}

		if ns.Name == "default" {
			free = ns.DataDiskFreespace
		}

		reserved += ns.DataLimit
	}

	return
}

func ipsToString(ips []net.IP) []string {
	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		result = append(result, ip.String())
	}

	return result
}

// isPublic check if ip is a IPv6 public address
func isPublic(ip net.IP) bool {
	if ip.To4() != nil {
		return false
	}

	return !(ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast())
}

// isPublic check if ip is a part of the yggdrasil 200::/7 range
var yggNet = net.IPNet{
	IP:   net.ParseIP("200::"),
	Mask: net.CIDRMask(7, 128),
}

var myceliumNet = net.IPNet{
	IP:   net.ParseIP("400::"),
	Mask: net.CIDRMask(7, 128),
}

func isYgg(ip net.IP) bool {
	return yggNet.Contains(ip)
}

func isMycelium(ip net.IP) bool {
	return myceliumNet.Contains(ip)
}

// InitializeZDB makes sure all required zdbs are running
func (p *Manager) Initialize(ctx context.Context) error {
	var (
		storage  = stubs.NewStorageModuleStub(p.zbus)
		contmod  = stubs.NewContainerModuleStub(p.zbus)
		network  = stubs.NewNetworkerStub(p.zbus)
		flistmod = stubs.NewFlisterStub(p.zbus)
	)
	// fetching extected hash
	log.Debug().Msg("fetching flist hash")
	expected, err := flistmod.FlistHash(ctx, zdbFlistURL)
	if err != nil {
		log.Error().Err(err).Msg("could not load expected flist hash")
		return err
	}

	devices, err := storage.Devices(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to list allocated zdb devices")
	}

	log.Debug().Msgf("alloced devices for zdb: %+v", devices)
	poolNames := make(map[string]pkg.Device)
	for _, device := range devices {
		poolNames[device.ID] = device
	}

	containers, err := contmod.List(ctx, zdbContainerNS)
	if err != nil {
		return errors.Wrap(err, "failed to list running zdb container")
	}

	for _, container := range containers {
		if err := p.upgradeRuntime(ctx, expected, container); err != nil {
			log.Error().Err(err).Msg("failed to upgrade running zdb container")
		}

		log.Debug().Str("container", string(container)).Msg("enusreing zdb network setup")
		_, err := network.EnsureZDBPrepare(ctx, poolNames[string(container)].ID)
		if err != nil {
			log.Error().Err(err).Msg("failed to prepare zdb network")
		}

		delete(poolNames, string(container))
	}

	// do we still have allocated pools that does not have associated zdbs.
	for _, device := range poolNames {
		log.Debug().Str("device", device.Path).Msg("starting zdb")
		if _, err := p.ensureZdbContainer(ctx, device); err != nil {
			log.Error().Err(err).Str("pool", device.ID).Msg("failed to create zdb container associated with pool")
		}
	}
	return nil
}

func (p *Manager) upgradeRuntime(ctx context.Context, expected string, container pkg.ContainerID) error {
	var (
		flistmod = stubs.NewFlisterStub(p.zbus)
		contmod  = stubs.NewContainerModuleStub(p.zbus)
	)
	continfo, err := contmod.Inspect(ctx, zdbContainerNS, container)
	if err != nil {
		return err
	}

	hash, err := flistmod.HashFromRootPath(ctx, continfo.RootFS)
	if err != nil {
		return errors.Wrap(err, "could not find container running flist hash")
	}

	log.Debug().Str("hash", hash).Msg("running container hash")
	if hash == expected {
		return nil
	}

	log.Info().Str("id", string(container)).Msg("restarting container, update found")

	// extracting required informations
	volumeid := continfo.Name // VolumeID is the Container Name
	volumepath := ""          // VolumePath is /data mount on the container
	socketdir := ""           // SocketDir is /socket on the container
	zdbcmd := continfo.Entrypoint
	netns := continfo.Network.Namespace

	log.Info().Str("id", volumeid).Str("path", volumepath).Msg("rebuild zdb container")

	for _, mnt := range continfo.Mounts {
		if mnt.Target == zdbContainerDataMnt {
			volumepath = mnt.Source
		}

		if mnt.Target == "/socket" {
			socketdir = mnt.Source
		}
	}

	if volumepath == "" {
		return fmt.Errorf("could not grab container /data mountpoint")
	}

	// stopping running zdb

	if err := contmod.Delete(ctx, zdbContainerNS, container); err != nil {
		return errors.Wrap(err, "could not stop running zdb container")
	}

	// cleanup old containers rootfs
	if err = flistmod.Unmount(ctx, volumeid); err != nil {
		log.Error().Err(err).Str("path", continfo.RootFS).Msgf("failed to unmount old zdb container")
	}

	// restarting zdb

	// mount the new flist
	rootfs, err := p.zdbRootFS(ctx)
	if err != nil {
		return errors.Wrap(err, "could not initialize zdb rootfs")
	}

	// respawn the container
	err = p.zdbRun(ctx, volumeid, rootfs, zdbcmd, netns, volumepath, socketdir)
	if err != nil {
		log.Error().Err(err).Msg("could not restart zdb container")

		if err = flistmod.Unmount(ctx, volumeid); err != nil {
			log.Error().Err(err).Str("path", rootfs).Msgf("failed to unmount zdb container")
		}
	}

	return nil
}
