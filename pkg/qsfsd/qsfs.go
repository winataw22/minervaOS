package qsfsd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	qsfsFlist             = "https://hub.grid.tf/tf-autobuilder/qsfs-0.2.0-rc2.flist"
	qsfsContainerNS       = "qsfs"
	qsfsRootFsPropagation = pkg.RootFSPropagationSlave
	zstorSocket           = "/var/run/zstor.sock"
	zstorZDBFSMountPoint  = "/mnt" // hardcoded in the container
	zstorMetricsPort      = 9100
	zstorZDBDataDirPath   = "/data/data/zdbfs-data"
	tombstonesDir         = "tombstones"
)

type QSFS struct {
	cl zbus.Client

	mountsPath     string
	tombstonesPath string
}

type zstorConfig struct {
	zos.QuantumSafeFSConfig
	ZDBDataDirPath  string `toml:"zdb_data_dir_path"`
	Socket          string `toml:"socket"`
	MetricsPort     uint32 `toml:"prometheus_port"`
	ZDBFSMountpoint string `toml:"zdbfs_mountpoint"`
}

func New(ctx context.Context, cl zbus.Client, root string) (pkg.QSFSD, error) {
	mountPath := filepath.Join(root, "mounts")
	err := os.MkdirAll(mountPath, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make qsfs mounts dir")
	}
	tombstonesPath := filepath.Join(root, tombstonesDir)
	err = os.MkdirAll(tombstonesPath, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make qsfs tombstones dir")
	}

	qsfs := &QSFS{
		cl:             cl,
		mountsPath:     mountPath,
		tombstonesPath: tombstonesPath,
	}
	if err := qsfs.migrateTombstones(ctx, cl); err != nil {
		return nil, err
	}
	go qsfs.periodicCleanup(ctx)
	return qsfs, nil
}

func (q *QSFS) migrateTombstones(ctx context.Context, cl zbus.Client) error {
	contd := stubs.NewContainerModuleStub(q.cl)
	containers, err := contd.List(ctx, qsfsContainerNS)
	if err != nil {
		return errors.Wrap(err, "couldn't list qsfs containers")
	}
	for _, contID := range containers {
		marked, err := q.isOldMarkedForDeletion(ctx, string(contID))
		if err != nil {
			log.Error().Err(err).Str("id", string(contID)).Msg("failed to check container old mark")
		}
		if marked {
			if err := q.markDelete(ctx, string(contID)); err != nil {
				log.Error().Err(err).Str("id", string(contID)).Msg("failed to mark container for deletion")
			}
		}
	}
	return nil
}

func setQSFSDefaults(cfg *zos.QuantumSafeFS) zstorConfig {
	return zstorConfig{
		QuantumSafeFSConfig: cfg.Config,
		Socket:              zstorSocket,
		MetricsPort:         zstorMetricsPort,
		ZDBFSMountpoint:     zstorZDBFSMountPoint,
		ZDBDataDirPath:      zstorZDBDataDirPath,
	}
}

func (q *QSFS) Mount(wlID string, cfg zos.QuantumSafeFS) (info pkg.QSFSInfo, err error) {
	defer func() {
		if err != nil {
			q.Unmount(wlID)
		}
	}()
	zstorConfig := setQSFSDefaults(&cfg)
	networkd := stubs.NewNetworkerStub(q.cl)
	flistd := stubs.NewFlisterStub(q.cl)
	contd := stubs.NewContainerModuleStub(q.cl)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	t := time.Now()
	defer cancel()
	marked, _ := q.isMarkedForDeletion(ctx, wlID)
	if marked {
		return info, errors.New("qsfs marked for deletion, try again later")
	}
	netns, yggIP, err := networkd.QSFSPrepare(ctx, wlID)
	if err != nil {
		return info, errors.Wrap(err, "failed to prepare qsfs")
	}
	flistPath, err := flistd.Mount(ctx, wlID, qsfsFlist, pkg.MountOptions{
		ReadOnly: false,
		Limit:    cfg.Cache,
	})
	if err != nil {
		err = errors.Wrap(err, "failed to mount qsfs flist")
		return
	}
	if lerr := q.writeQSFSConfig(flistPath, zstorConfig); lerr != nil {
		err = errors.Wrap(lerr, "couldn't write qsfs config")
		return
	}
	mountPath := q.mountPath(wlID)
	err = q.prepareMountPath(mountPath)
	if err != nil {
		err = errors.Wrap(err, "failed to prepare mount path")
		return
	}
	cont := pkg.Container{
		Name:        wlID,
		RootFS:      flistPath,
		Entrypoint:  "/sbin/zinit init --container",
		Interactive: false,
		Network:     pkg.NetworkInfo{Namespace: netns},
		Memory:      gridtypes.Gigabyte,
		Mounts: []pkg.MountInfo{
			{
				Source: mountPath,
				Target: "/mnt",
			},
		},
		Elevated: true,
		// the default is rslave which recursively sets all mountpoints to slave
		// we don't care about the rootfs propagation, it just has to be non-recursive
		RootFsPropagation: qsfsRootFsPropagation,
	}
	_, err = contd.Run(
		ctx,
		qsfsContainerNS,
		cont,
	)
	log.Debug().Str("duration", time.Since(t).String()).Msg("time before waiting for qsfs mountpoint")
	if lerr := q.waitUntilMounted(ctx, mountPath); lerr != nil {
		logs, containerErr := contd.Logs(ctx, qsfsContainerNS, wlID)
		if containerErr != nil {
			log.Error().Err(containerErr).Msg("Failed to read container logs")
		}
		err = errors.Wrapf(lerr, "Container Logs:\n%s", logs)
		return
	}
	log.Debug().Str("duration", time.Since(t).String()).Msg("waiting for qsfs deployment took")
	info.Path = mountPath
	info.MetricsEndpoint = fmt.Sprintf("http://[%s]:%d/metrics", yggIP, zstorMetricsPort)

	return
}

func (f *QSFS) waitUntilMounted(ctx context.Context, path string) error {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			mounted, err := f.isMounted(path)
			if err != nil {
				return errors.Wrap(err, "failed to check if zdbfs is mounted")
			}
			if mounted {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("waiting for zdbfs mount %s timedout: context cancelled", path)
		}
	}

}

func (f *QSFS) isMounted(path string) (bool, error) {
	output, err := exec.Command("findmnt", "-J", path).Output()
	if err, ok := err.(*exec.ExitError); ok && err != nil {
		if err.ExitCode() == 1 {
			return false, nil
		}
	}
	var result struct {
		Filesystems []struct {
			Fstype string `json:"fstype"`
		} `json:"filesystems"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return false, errors.Wrap(err, "failed to parse findmnt output")
	}
	for _, fs := range result.Filesystems {
		if fs.Fstype == "fuse.zdbfs" {
			return true, nil
		}
	}
	return false, nil
}

func (q *QSFS) UpdateMount(wlID string, cfg zos.QuantumSafeFS) (pkg.QSFSInfo, error) {
	var info pkg.QSFSInfo
	zstorConfig := setQSFSDefaults(&cfg)
	networkd := stubs.NewNetworkerStub(q.cl)
	flistd := stubs.NewFlisterStub(q.cl)
	contd := stubs.NewContainerModuleStub(q.cl)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	marked, err := q.isMarkedForDeletion(ctx, wlID)
	if marked {
		return info, errors.New("already marked for deletion")
	}
	if err != nil {
		return info, errors.Wrap(err, "failed to check deletion mark")
	}
	yggIP, err := networkd.QSFSYggIP(ctx, wlID)
	if err != nil {
		return info, errors.Wrap(err, "failed to get ygg ip")
	}
	flistPath, err := flistd.UpdateMountSize(ctx, wlID, cfg.Cache)
	if err != nil {
		return info, errors.Wrap(err, "failed to get qsfs flist mountpoint")
	}
	if err := q.writeQSFSConfig(flistPath, zstorConfig); err != nil {
		return info, errors.Wrap(err, "couldn't write qsfs config")
	}
	mountPath := q.mountPath(wlID)

	if err := contd.Exec(ctx, qsfsContainerNS, wlID, 10*time.Second, "/sbin/zinit", "kill", "zstor", "SIGINT"); err != nil {
		return info, errors.Wrap(err, "failed to restart zstor process")
	}
	info.Path = mountPath
	info.MetricsEndpoint = fmt.Sprintf("http://[%s]:%d/metrics", yggIP, zstorMetricsPort)
	return info, nil
}

func (q *QSFS) SignalDelete(wlID string) error {
	contd := stubs.NewContainerModuleStub(q.cl)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	marked, err := q.isMarkedForDeletion(ctx, wlID)

	if marked {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "failed to check deletion mark")
	}
	if err := q.markDelete(ctx, wlID); err != nil {
		// container dead, no need to continue
		return err
	}
	if err := contd.SignalDelete(ctx, qsfsContainerNS, pkg.ContainerID(wlID)); err != nil {
		return errors.Wrap(err, "couldn't stop qsfs container")
	}
	return nil
}

func (q *QSFS) mountPath(wlID string) string {
	return filepath.Join(q.mountsPath, wlID)
}

func (q *QSFS) prepareMountPath(path string) error {
	if err := os.MkdirAll(path, 0644); err != nil {
		return err
	}

	// container mounts doesn't appear on the host if this is not mounted
	if err := syscall.Mount(path, path, "bind", syscall.MS_BIND, ""); err != nil {
		return err
	}
	if err := syscall.Mount("", path, "", syscall.MS_SHARED, ""); err != nil {
		return err
	}
	return nil
}

func (q *QSFS) writeQSFSConfig(root string, cfg zstorConfig) error {
	cfgPath := filepath.Join(root, "data/zstor.toml")
	f, err := os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "couldn't open zstor config file")
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return errors.Wrap(err, "failed to convert config to yaml")
	}

	return nil
}
