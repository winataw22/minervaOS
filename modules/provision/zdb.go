package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zosv2/modules/zdb"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	nwmod "github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/stubs"
)

const (
	// TODO: make this configurable
	zdbFlistURL    = "https://hub.grid.tf/tf-autobuilder/threefoldtech-0-db-development.flist"
	zdbContainerNS = "zdb"
	zdbPort        = 9900
)

// ZDB namespace creation info
type ZDB struct {
	Size     uint64             `json:"size"`
	Mode     modules.ZDBMode    `json:"mode"`
	Password string             `json:"password"`
	DiskType modules.DeviceType `json:"disk_type"`
	Public   bool               `json:"public"`
}

// ZDBResult is the information return to the BCDB
// after deploying a 0-db namespace
type ZDBResult struct {
	Namespace string
	IP        string
	Port      uint
}

// ZDBMapping is a helper struct that allow to keep
// a mapping between a 0-db namespace and the container ID
// in which it lives
type ZDBMapping struct {
	m map[string]string
	sync.RWMutex
}

// Get returns the container ID where namespace lives
// if the namespace is not found an empty string and false is returned
func (z *ZDBMapping) Get(namespace string) (string, bool) {
	z.RLock()
	defer z.RUnlock()

	if z.m == nil {
		return "", false
	}

	id, ok := z.m[namespace]
	return id, ok
}

// Set saves the mapping between the namespace and a container ID
func (z *ZDBMapping) Set(namespace, container string) {
	z.Lock()
	defer z.Unlock()

	if z.m == nil {
		z.m = make(map[string]string, 1)
	}

	z.m[namespace] = container
}

func zdbProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {

	var (
		client = GetZBus(ctx)
		zdbMap = GetZDBMapping(ctx)

		container = stubs.NewContainerModuleStub(client)
		storage   = stubs.NewZDBAllocaterStub(client)
		network   = stubs.NewNetworkerStub(client)

		nsID        = reservation.ID
		config      ZDB
		containerIP net.IP
	)
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return nil, errors.Wrap(err, "failed to decode reservation schema")
	}

	// verify if the namespace isn't already deployed
	containerID, ok := zdbMap.Get(nsID)
	log.Debug().Msg("zdb provision, check if namespace already deployed")
	if ok {
		c, err := container.Inspect(zdbContainerNS, modules.ContainerID(containerID))
		if err != nil {
			return nil, err
		}

		ips, err := network.Addrs(nwmod.ZDBIface, c.Network.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get 0-db container IP")
		}
		for _, ip := range ips {
			if isPublic(ip) {
				containerIP = ip
				break
			}
		}
		if containerIP == nil {
			return nil, errors.Wrap(err, "failed to get 0-db container IP")
		}

		return ZDBResult{
			Namespace: nsID,
			IP:        containerIP.String(),
			Port:      zdbPort,
		}, nil
	}

	// if we reached here, we need to create the 0-db namespace
	log.Debug().Msg("try to allocate storage")
	containerID, vPath, err := storage.Allocate(config.DiskType, config.Size*gigabyte, config.Mode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to allocate storage")
	}

	slog := log.With().
		Str("containerID", containerID).
		Logger()

	// check if there is already a 0-DB running container on this volume
	slog.Debug().Msg("check if container already exist on this volume")
	_, err = container.Inspect(zdbContainerNS, modules.ContainerID(containerID))

	//FIXME: find a better way then parsing error content
	// Here we loose the error value cause the error comes from zbus
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return nil, errors.Wrapf(err, "failed to check if 0-db container already exists")
	}

	//FIXME: find a better way then parsing error content
	if err != nil && strings.Contains(err.Error(), "not found") {
		slog.Info().Msgf("0-db container not found, start creation")

		containerIP, err = createZdbContainer(ctx, containerID, config.Mode, vPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create 0-db container")
		}

		// save in which container this namespace will be running
		zdbMap.Set(nsID, string(containerID))
	}

	// at this point there should always be a container running for this 0-db
	slog.Debug().Msg("check if we already know the ip of the container")
	if containerIP == nil {
		slog.Debug().Msg("not known, check it from the container")
		c, err := container.Inspect(zdbContainerNS, modules.ContainerID(containerID))
		if err != nil {
			return nil, errors.Wrap(err, "failed to have a 0-db container running")
		}

		ips, err := network.Addrs(nwmod.ZDBIface, c.Network.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get 0-db container IP")
		}
		for _, ip := range ips {
			if isPublic(ip) {
				containerIP = ip
				break
			}
		}
		if containerIP == nil {
			return nil, errors.Wrap(err, "failed to get 0-db container IP")
		}
	}
	slog.Info().IPAddr("ip", containerIP).Msg("container IP found")

	slog.Info().
		Str("name", nsID).
		Str("container", containerID).
		Msg("create 0-db namespace")

	if err := createZDBNamespace(containerID, nsID, config); err != nil {
		return nil, err
	}

	return ZDBResult{
		Namespace: nsID,
		IP:        containerIP.String(),
		Port:      zdbPort,
	}, nil
}

func createZdbContainer(ctx context.Context, name string, mode modules.ZDBMode, volumePath string) (net.IP, error) {
	var (
		client = GetZBus(ctx)

		cont    = stubs.NewContainerModuleStub(client)
		flist   = stubs.NewFlisterStub(client)
		network = stubs.NewNetworkerStub(client)

		containerIP net.IP

		slog = log.With().Str("containerID", name).Logger()
	)

	slog.Debug().Str("flist", zdbFlistURL).Msg("mounting flist")
	rootFS, err := flist.Mount(zdbFlistURL, "")
	if err != nil {
		return nil, err
	}

	cleanup := func() {
		if err := cont.Delete(zdbContainerNS, modules.ContainerID(name)); err != nil {
			slog.Error().Str("container", name).Err(err).Msg("failed to delete 0-db container")
		}

		if err := flist.Umount(rootFS); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}
	}

	// create the network namespace and macvlan for the 0-db container
	netNsName, err := network.ZDBPrepare()
	if err != nil {
		if err := flist.Umount(rootFS); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}

		return nil, err
	}

	socketDir := socketDir(name)
	if err := os.MkdirAll(socketDir, 0550); err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("/bin/zdb --data /data --index /data --mode %s  --listen :: --port %d --socket /socket/zdb.sock --dualnet", string(mode), zdbPort)
	_, err = cont.Run(
		zdbContainerNS,
		modules.Container{
			Name:        name,
			RootFS:      rootFS,
			Entrypoint:  cmd,
			Interactive: false,
			Network:     modules.NetworkInfo{Namespace: netNsName},
			Mounts: []modules.MountInfo{
				{
					Source:  volumePath,
					Target:  "/data",
					Type:    "none",
					Options: []string{"bind"},
				},
				{
					Source:  socketDir,
					Target:  "/socket",
					Type:    "none",
					Options: []string{"bind"},
				},
			},
		})
	if err != nil {
		if err := flist.Umount(rootFS); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}
		return nil, errors.Wrap(err, "failed to create container")
	}

	getIP := func() error {
		ips, err := network.Addrs(nwmod.ZDBIface, netNsName)
		if err != nil {
			slog.Debug().Err(err).Msg("not ip public found, waiting")
			return err
		}
		for _, ip := range ips {
			if isPublic(ip) {
				slog.Debug().IPAddr("ip", ip).Msg("0-db container public ip found")
				containerIP = ip
				return nil
			}
		}
		return fmt.Errorf("not up public found, waiting")
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Minute * 2
	if err := backoff.RetryNotify(getIP, bo, nil); err != nil {
		cleanup()
		return nil, errors.Wrapf(err, "failed to get an IP for 0-db container %s", name)
	}

	slog.Info().
		IPAddr("container IP", containerIP).
		Str("name", name).
		Msgf("0-db container created")
	return containerIP, nil
}

func createZDBNamespace(containerID, nsID string, config ZDB) error {
	zdbCl := zdb.New()

	addr := fmt.Sprintf("unix://%s/zdb.sock", socketDir(containerID))
	if err := zdbCl.Connect(addr); err != nil {
		return errors.Wrapf(err, "failed to connect to 0-db at %s", addr)
	}
	defer zdbCl.Close()

	if err := zdbCl.CreateNamespace(nsID); err != nil {
		return errors.Wrapf(err, "failed to create namespace in 0-db at %s", addr)
	}

	if config.Password != "" {
		if err := zdbCl.NamespaceSetPassword(nsID, config.Password); err != nil {
			return errors.Wrapf(err, "failed to set password namespace %s in 0-db at %s", nsID, addr)
		}
	}

	if err := zdbCl.NamespaceSetPublic(nsID, config.Public); err != nil {
		return errors.Wrapf(err, "failed to make namespace %s public in 0-db at %s", nsID, addr)
	}

	if err := zdbCl.NamespaceSetSize(nsID, config.Size*gigabyte); err != nil {
		return errors.Wrapf(err, "failed to set size on namespace %s in 0-db at %s", nsID, addr)
	}

	return nil
}

func zdbDecommission(ctx context.Context, reservation *Reservation) error {
	var (
		client = GetZBus(ctx)
		zdbMap = GetZDBMapping(ctx)

		container = stubs.NewContainerModuleStub(client)
		storage   = stubs.NewZDBAllocaterStub(client)

		config ZDB
		nsID   = reservation.ID

		slog = log.With().Str("namespace", nsID).Logger()
	)

	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	containerID, ok := zdbMap.Get(nsID)
	if !ok {
		return nil
	}

	addr := fmt.Sprintf("unix://%s/zdb.sock", socketDir(containerID))
	slog.Debug().Str("addr", addr).Msg("connect to 0-db and delete namespace")

	zdbCl := zdb.New()
	if err := zdbCl.Connect(addr); err != nil {
		return errors.Wrapf(err, "failed to connect to 0-db at %s", addr)
	}
	defer zdbCl.Close()

	if err := zdbCl.DeleteNamespace(nsID); err != nil {
		return errors.Wrapf(err, "failed to delete namespace in 0-db at %s", addr)
	}

	c, err := container.Inspect(zdbContainerNS, modules.ContainerID(containerID))
	if err != nil {
		return err
	}

	if len(c.Mounts) < 1 {
		return fmt.Errorf("no mountpoint find in 0-db container, cannot reclaim storage")
	}
	volume := filepath.Base(c.Mounts[0].Source)

	if err := storage.Claim(volume, config.Size); err != nil {
		return errors.Wrapf(err, "failed to reclaim storage on volume %s", volume)
	}

	slog.Info().Msgf("0-db namespace %s deleted", nsID)
	return nil
}

func zdbConnectionURL(ip string, port uint16) string {
	return fmt.Sprintf("tcp://[%s]:%d", ip, port)
}

func socketDir(containerID string) string {
	return fmt.Sprintf("/var/run/zdb_%s", containerID)
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
