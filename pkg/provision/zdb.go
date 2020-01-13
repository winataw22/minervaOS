package provision

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/zdb"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	nwmod "github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	// TODO: make this configurable
	zdbFlistURL    = "https://hub.grid.tf/tf-autobuilder/threefoldtech-0-db-development.flist"
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
	IP        string
	Port      uint
}

func zdbProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {
	return zdbProvisionImpl(ctx, reservation)
}

func decryptPassword(client zbus.Client, password string) (string, error) {
	if len(password) == 0 {
		return "", nil
	}
	identity := stubs.NewIdentityManagerStub(client)

	bytes, err := hex.DecodeString(password)
	if err != nil {
		return "", err
	}

	out, err := identity.Decrypt(bytes)
	return string(out), err
}

func zdbProvisionImpl(ctx context.Context, reservation *Reservation) (ZDBResult, error) {
	var (
		client  = GetZBus(ctx)
		storage = stubs.NewZDBAllocaterStub(client)

		nsID        = reservation.ID
		config      ZDB
		containerIP net.IP
	)
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return ZDBResult{}, errors.Wrap(err, "failed to decode reservation schema")
	}

	var err error
	config.PlainPassword, err = decryptPassword(client, config.Password)
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

	cont, err := ensureZdbContainer(ctx, allocation, config.Mode)
	if err != nil {
		return ZDBResult{}, err
	}

	containerIP, err = getZDBIP(ctx, cont)
	if err != nil {
		return ZDBResult{}, err
	}

	// this call will actually configure the namespace in zdb and set the password
	if err := createZDBNamespace(containerID, nsID, config); err != nil {
		return ZDBResult{}, err
	}

	return ZDBResult{
		Namespace: nsID,
		IP:        containerIP.String(),
		Port:      zdbPort,
	}, nil
}

func ensureZdbContainer(ctx context.Context, allocation pkg.Allocation, mode pkg.ZDBMode) (pkg.Container, error) {
	var (
		client    = GetZBus(ctx)
		container = stubs.NewContainerModuleStub(client)
	)

	name := pkg.ContainerID(allocation.VolumeID)

	cont, err := container.Inspect(zdbContainerNS, name)
	if err != nil && strings.Contains(err.Error(), "not found") {
		// container not found, create one
		if err := createZdbContainer(ctx, allocation, mode); err != nil {
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

func createZdbContainer(ctx context.Context, allocation pkg.Allocation, mode pkg.ZDBMode) error {
	var (
		name       = pkg.ContainerID(allocation.VolumeID)
		volumePath = allocation.VolumePath

		client = GetZBus(ctx)

		cont    = stubs.NewContainerModuleStub(client)
		flist   = stubs.NewFlisterStub(client)
		network = stubs.NewNetworkerStub(client)

		slog = log.With().Str("containerID", string(name)).Logger()
	)

	hw := ifaceutil.HardwareAddrFromInputBytes([]byte(allocation.VolumeID))

	slog.Debug().Str("flist", zdbFlistURL).Msg("mounting flist")
	rootFS, err := flist.Mount(zdbFlistURL, "", pkg.MountOptions{
		Limit:    10,
		ReadOnly: false,
	})
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

		return err
	}

	socketDir := socketDir(name)
	if err := os.MkdirAll(socketDir, 0550); err != nil {
		return err
	}

	cmd := fmt.Sprintf("/bin/zdb --data /data --index /data --mode %s  --listen :: --port %d --socket /socket/zdb.sock --dualnet", string(mode), zdbPort)
	_, err = cont.Run(
		zdbContainerNS,
		pkg.Container{
			Name:        string(name),
			RootFS:      rootFS,
			Entrypoint:  cmd,
			Interactive: false,
			Network:     pkg.NetworkInfo{Namespace: netNsName},
			Mounts: []pkg.MountInfo{
				{
					Source: volumePath,
					Target: "/data",
				},
				{
					Source: socketDir,
					Target: "/socket",
				},
			},
		})

	if err != nil {
		cleanup()
		return errors.Wrap(err, "failed to create container")
	}

	cl := zdbConnection(name)
	defer cl.Close()

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Minute * 2
	bo.MaxElapsedTime = time.Minute * 2

	if err := backoff.RetryNotify(cl.Connect, bo, func(err error, d time.Duration) {
		log.Debug().Err(err).Str("duration", d.String()).Msg("waiting for zdb to start")
	}); err != nil {
		cleanup()
		return errors.Wrapf(err, "failed to establish connection to zdb")
	}

	return nil
}

func getZDBIP(ctx context.Context, container pkg.Container) (containerIP net.IP, err error) {
	var (
		client  = GetZBus(ctx)
		network = stubs.NewNetworkerStub(client)

		slog = log.With().Str("containerID", container.Name).Logger()
	)

	getIP := func() error {
		ips, err := network.Addrs(nwmod.ZDBIface, container.Network.Namespace)
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
	bo.MaxElapsedTime = time.Minute * 2

	if err := backoff.RetryNotify(getIP, bo, func(err error, d time.Duration) {
		log.Debug().Err(err).Str("duration", d.String()).Msg("failed to get zdb public IP")
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to get an IP for 0-db container %s", container.Name)
	}

	slog.Info().
		IPAddr("container IP", containerIP).
		Str("name", container.Name).
		Msgf("0-db container created")
	return containerIP, nil
}

func createZDBNamespace(containerID pkg.ContainerID, nsID string, config ZDB) error {
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

func zdbDecommission(ctx context.Context, reservation *Reservation) error {
	var (
		client  = GetZBus(ctx)
		storage = stubs.NewZDBAllocaterStub(client)

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

	_, err = ensureZdbContainer(ctx, allocation, config.Mode)
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

	return nil
}

func zdbConnectionURL(ip string, port uint16) string {
	return fmt.Sprintf("tcp://[%s]:%d", ip, port)
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

func findDataVolume(mounts []pkg.MountInfo) (string, error) {
	for _, mount := range mounts {
		if mount.Target == "/data" {
			return filepath.Base(mount.Source), nil
		}
	}
	return "", fmt.Errorf("data volume not found")
}
