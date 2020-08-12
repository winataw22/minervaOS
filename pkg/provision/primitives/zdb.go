package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
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
	zdbFlistURL    = "https://hub.grid.tf/tf-autobuilder/threefoldtech-0-db-development-b5155357d5.flist"
	zdbContainerNS = "zdb"
	zdbPort        = 9900
)

// ZDB namespace creation info
type ZDB struct {
	Size     uint64         `json:"size"`
	Mode     pkg.ZDBMode    `json:"mode"`
	Password string         `json:"password"`
	DiskType pkg.DeviceType `json:"disk_type"`
	Public   bool           `json:"public"`

	PlainPassword string `json:"-"`
}

// ZDBResult is the information return to the BCDB
// after deploying a 0-db namespace
type ZDBResult struct {
	Namespace string
	IPs       []string
	Port      uint
}

func (p *Provisioner) zdbProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.zdbProvisionImpl(ctx, reservation)
}

func (p *Provisioner) zdbProvisionImpl(ctx context.Context, reservation *provision.Reservation) (ZDBResult, error) {
	var (
		storage = stubs.NewZDBAllocaterStub(p.zbus)

		nsID   = reservation.ID
		config ZDB
	)
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return ZDBResult{}, errors.Wrap(err, "failed to decode reservation schema")
	}

	var err error
	config.PlainPassword, err = decryptSecret(p.zbus, config.Password)
	if err != nil {
		return ZDBResult{}, errors.Wrap(err, "failed to decrypt namespace password")
	}

	// if we reached here, we need to create the 0-db namespace
	log.Debug().Msg("allocating storage for namespace")
	allocation, err := storage.Allocate(nsID, config.DiskType, config.Size*gigabyte, config.Mode)
	if err != nil {
		return ZDBResult{}, errors.Wrap(err, "failed to allocate storage")
	}

	containerID := pkg.ContainerID(allocation.VolumeID)

	cont, err := p.ensureZdbContainer(ctx, allocation, config.Mode)
	if err != nil {
		return ZDBResult{}, errors.Wrapf(err, "failed to ensure zdb containe running")
	}

	containerIPs, err := p.waitZDBIPs(ctx, nwmod.ZDBIface, cont.Network.Namespace)
	if err != nil {
		return ZDBResult{}, errors.Wrap(err, "failed to find IP address on zdb0 interface")
	}
	log.Warn().Msgf("ip for zdb containers %s", containerIPs)

	// this call will actually configure the namespace in zdb and set the password
	if err := p.createZDBNamespace(containerID, nsID, config); err != nil {
		return ZDBResult{}, errors.Wrap(err, "failed to create zdb namespace")
	}

	return ZDBResult{
		Namespace: nsID,
		IPs: func() []string {
			ips := make([]string, len(containerIPs))
			for i, ip := range containerIPs {
				ips[i] = ip.String()
			}
			return ips
		}(),
		Port: zdbPort,
	}, nil
}

func (p *Provisioner) ensureZdbContainer(ctx context.Context, allocation pkg.Allocation, mode pkg.ZDBMode) (pkg.Container, error) {
	var container = stubs.NewContainerModuleStub(p.zbus)

	name := pkg.ContainerID(allocation.VolumeID)

	cont, err := container.Inspect(zdbContainerNS, name)
	if err != nil && strings.Contains(err.Error(), "not found") {
		// container not found, create one
		if err := p.createZdbContainer(ctx, allocation, mode); err != nil {
			return cont, err
		}
		cont, err = container.Inspect(zdbContainerNS, name)
		if err != nil {
			return pkg.Container{}, err
		}
	} else if err != nil {
		// other error
		return pkg.Container{}, err
	}

	return cont, nil

}

func (p *Provisioner) zdbRootFS() (string, error) {
	var flist = stubs.NewFlisterStub(p.zbus)
	var err error
	var rootFS string

	for _, typ := range []pkg.DeviceType{pkg.HDDDevice, pkg.SSDDevice} {
		rootFS, err = flist.Mount(zdbFlistURL, "", pkg.MountOptions{
			Limit:    10,
			ReadOnly: false,
			Type:     typ,
		})

		if err != nil {
			log.Error().Err(err).Msgf("failed to allocate rootfs for zdb container (type: '%s'): %s", typ, err)
		}

		if err == nil {
			break
		}
	}

	if err != nil {
		return "", errors.Wrap(err, "failed to allocate rootfs for zdb container")
	}

	return rootFS, nil
}

func (p *Provisioner) createZdbContainer(ctx context.Context, allocation pkg.Allocation, mode pkg.ZDBMode) error {
	var (
		name       = pkg.ContainerID(allocation.VolumeID)
		cont       = stubs.NewContainerModuleStub(p.zbus)
		flist      = stubs.NewFlisterStub(p.zbus)
		volumePath = allocation.VolumePath
		network    = stubs.NewNetworkerStub(p.zbus)

		slog = log.With().Str("containerID", string(name)).Logger()
	)

	hw := ifaceutil.HardwareAddrFromInputBytes([]byte(allocation.VolumeID))

	slog.Debug().Str("flist", zdbFlistURL).Msg("mounting flist")

	rootFS, err := p.zdbRootFS()
	if err != nil {
		return err
	}

	cleanup := func() {
		if err := cont.Delete(zdbContainerNS, name); err != nil {
			slog.Error().Err(err).Msg("failed to delete 0-db container")
		}

		if err := flist.Umount(rootFS); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}
	}

	// create the network namespace and macvlan for the 0-db container
	netNsName, err := network.ZDBPrepare(hw)
	if err != nil {
		if err := flist.Umount(rootFS); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}

		return errors.Wrap(err, "failed to prepare zdb network")
	}

	socketDir := socketDir(name)
	if err := os.MkdirAll(socketDir, 0550); err != nil {
		return errors.Wrapf(err, "failed to create directory: %s", socketDir)
	}

	cmd := fmt.Sprintf("/bin/zdb --data /data --index /data --mode %s  --listen :: --port %d --socket /socket/zdb.sock --dualnet", string(mode), zdbPort)

	err = p.zdbRun(string(name), rootFS, cmd, netNsName, volumePath, socketDir)
	if err != nil {
		cleanup()
		return errors.Wrap(err, "failed to create container")
	}

	cl := zdbConnection(name)
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

func (p *Provisioner) zdbRun(name string, rootfs string, cmd string, netns string, volumepath string, socketdir string) error {
	var cont = stubs.NewContainerModuleStub(p.zbus)

	_, err := cont.Run(
		zdbContainerNS,
		pkg.Container{
			Name:        name,
			RootFS:      rootfs,
			Entrypoint:  cmd,
			Interactive: false,
			Network:     pkg.NetworkInfo{Namespace: netns},
			Mounts: []pkg.MountInfo{
				{
					Source: volumepath,
					Target: "/data",
				},
				{
					Source: socketdir,
					Target: "/socket",
				},
			},
		})

	return err
}

func (p *Provisioner) waitZDBIPs(ctx context.Context, ifaceName, namespace string) ([]net.IP, error) {
	var (
		network      = stubs.NewNetworkerStub(p.zbus)
		containerIPs []net.IP
	)

	getIP := func() error {

		ips, err := network.Addrs(ifaceName, namespace)
		if err != nil {
			log.Debug().Err(err).Msg("not ip public found, waiting")
			return err
		}

		var (
			public = false
			ygg    = false
		)
		containerIPs = containerIPs[:0]

		for _, ip := range ips {
			if isPublic(ip) && !isYgg(ip) {
				log.Warn().IPAddr("ip", ip).Msg("0-db container public ip found")
				public = true
				containerIPs = append(containerIPs, ip)
			}
			if isYgg(ip) {
				log.Warn().IPAddr("ip", ip).Msg("0-db container ygg ip found")
				ygg = true
				containerIPs = append(containerIPs, ip)
			}
		}

		log.Warn().Msgf("public %v ygg: %v", public, ygg)
		if public && ygg {
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
		return nil, errors.Wrapf(err, "failed to get an IP for interface %s", ifaceName)
	}

	return containerIPs, nil
}

func (p *Provisioner) createZDBNamespace(containerID pkg.ContainerID, nsID string, config ZDB) error {
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
	}

	if config.PlainPassword != "" {
		if err := zdbCl.NamespaceSetPassword(nsID, config.PlainPassword); err != nil {
			return errors.Wrapf(err, "failed to set password namespace %s in 0-db: %s", nsID, containerID)
		}
	}

	if err := zdbCl.NamespaceSetPublic(nsID, config.Public); err != nil {
		return errors.Wrapf(err, "failed to make namespace %s public in 0-db: %s", nsID, containerID)
	}

	if err := zdbCl.NamespaceSetSize(nsID, config.Size*gigabyte); err != nil {
		return errors.Wrapf(err, "failed to set size on namespace %s in 0-db: %s", nsID, containerID)
	}

	return nil
}

func (p *Provisioner) zdbDecommission(ctx context.Context, reservation *provision.Reservation) error {
	var (
		storage       = stubs.NewZDBAllocaterStub(p.zbus)
		storageClient = stubs.NewStorageModuleStub(p.zbus)

		config ZDB
		nsID   = reservation.ID
	)

	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	allocation, err := storage.Find(reservation.ID)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return nil
	} else if err != nil {
		return err
	}

	_, err = p.ensureZdbContainer(ctx, allocation, config.Mode)
	if err != nil {
		return errors.Wrap(err, "failed to find namespace zdb container")
	}

	containerID := pkg.ContainerID(allocation.VolumeID)

	zdbCl := zdbConnection(containerID)
	defer zdbCl.Close()
	if err := zdbCl.Connect(); err != nil {
		return errors.Wrapf(err, "failed to connect to 0-db: %s", containerID)
	}

	if err := zdbCl.DeleteNamespace(nsID); err != nil {
		return errors.Wrapf(err, "failed to delete namespace in 0-db: %s", containerID)
	}

	ns, err := zdbCl.Namespaces()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve zdb namespaces")
	}

	log.Info().Msgf("zdb has %d namespaces left", len(ns))

	// If there are no more namespaces left except for the default namespace, we can delete this subvolume
	if len(ns) == 1 && ns[0] == "default" {
		log.Info().Msg("decommissioning zdb container because there are no more namespaces left")
		err = p.deleteZdbContainer(containerID)
		if err != nil {
			return errors.Wrap(err, "failed to decommission zdb container")
		}

		log.Info().Msgf("deleting subvolumes of reservation: %s", allocation.VolumeID)
		// we also need to delete the flist volume
		return storageClient.ReleaseFilesystem(allocation.VolumeID)
	}

	return nil
}

func (p *Provisioner) deleteZdbContainer(containerID pkg.ContainerID) error {
	container := stubs.NewContainerModuleStub(p.zbus)
	flist := stubs.NewFlisterStub(p.zbus)
	// networkMgr := stubs.NewNetworkerStub(p.zbus)

	info, err := container.Inspect("zdb", containerID)
	if err == nil {
		if err := container.Delete("zdb", containerID); err != nil {
			return errors.Wrapf(err, "failed to delete container %s", containerID)
		}

		rootFS := info.RootFS
		if info.Interactive {
			rootFS, err = findRootFS(info.Mounts)
			if err != nil {
				return err
			}
		}

		if err := flist.Umount(rootFS); err != nil {
			return errors.Wrapf(err, "failed to unmount flist at %s", rootFS)
		}

	} else {
		log.Error().Err(err).Str("container", string(containerID)).Msg("failed to inspect container for decomission")
	}

	// TODO: delete network?

	return nil
}

func socketDir(containerID pkg.ContainerID) string {
	return fmt.Sprintf("/var/run/zdb_%s", containerID)
}

// we declare this method as a variable so we can
// mock it in testing.
var zdbConnection = func(id pkg.ContainerID) zdb.Client {
	socket := fmt.Sprintf("unix://%s/zdb.sock", socketDir(id))
	return zdb.New(socket)
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

func isYgg(ip net.IP) bool {
	return yggNet.Contains(ip)
}

func (p *Provisioner) upgradeRunningZdb(ctx context.Context) error {
	log.Info().Msg("checking for any outdated zdb running")

	flistmod := stubs.NewFlisterStub(p.zbus)
	contmod := stubs.NewContainerModuleStub(p.zbus)

	// Listing running zdb containers
	containers, err := contmod.List(zdbContainerNS)
	if err != nil {
		log.Error().Err(err).Msg("could not load containers list")
		return err
	}

	// fetching extected hash
	expected, err := flistmod.FlistHash(zdbFlistURL)
	if err != nil {
		log.Error().Err(err).Msg("could not load expected flist hash")
		return err
	}

	// Checking if containers are running latest zdb version
	for _, c := range containers {
		if c == "" {
			continue
		}

		log.Debug().Str("id", string(c)).Msg("inspecting container")

		continfo, err := contmod.Inspect(zdbContainerNS, c)

		if err != nil {
			log.Error().Err(err).Msg("could not inspect container")
			continue
		}

		hash, err := flistmod.HashFromRootPath(continfo.RootFS)
		if err != nil {
			log.Error().Err(err).Msg("could not find container running flist hash")
			continue
		}

		log.Debug().Str("hash", hash).Msg("running container hash")

		if expected != hash {
			log.Info().Str("id", string(c)).Msg("restarting container, update found")

			// extracting required informations
			volumeid := continfo.Name // VolumeID is the Container Name
			volumepath := ""          // VolumePath is /data mount on the container
			socketdir := ""           // SocketDir is /socket on the container
			zdbcmd := continfo.Entrypoint
			netns := continfo.Network.Namespace

			log.Info().Str("id", volumeid).Str("path", volumepath).Msg("rebuild zdb container")

			for _, mnt := range continfo.Mounts {
				if mnt.Target == "/data" {
					volumepath = mnt.Source
				}

				if mnt.Target == "/socket" {
					socketdir = mnt.Source
				}
			}

			if volumepath == "" {
				log.Error().Msg("could not grab container /data mountpoint")
				continue
			}

			// stopping running zdb
			err := contmod.Delete(zdbContainerNS, c)
			if err != nil {
				log.Error().Err(err).Msg("could not stop running zdb container")
				continue
			}

			// cleanup old containers rootfs
			if err = flistmod.Umount(continfo.RootFS); err != nil {
				log.Error().Err(err).Str("path", continfo.RootFS).Msgf("failed to unmount old zdb container")
			}

			// restarting zdb

			// mount the new flist
			rootfs, err := p.zdbRootFS()
			if err != nil {
				log.Error().Err(err).Msg("could not initialize zdb rootfs")
				continue
			}

			// respawn the container
			err = p.zdbRun(volumeid, rootfs, zdbcmd, netns, volumepath, socketdir)
			if err != nil {
				log.Error().Err(err).Msg("could not restart zdb container")

				if err = flistmod.Umount(rootfs); err != nil {
					log.Error().Err(err).Str("path", rootfs).Msgf("failed to unmount zdb container")
				}
			}
		}
	}

	return nil
}
