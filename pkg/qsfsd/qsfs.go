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
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	qsfsFlist             = "https://hub.grid.tf/azmy.3bot/qsfs.flist"
	qsfsContainerNS       = "qsfs"
	qsfsRootFsPropagation = pkg.RootFSPropagationSlave
)

type QSFS struct {
	cl zbus.Client

	mountsPath string
}

func New(ctx context.Context, cl zbus.Client, root string) (pkg.QSFSD, error) {
	mountPath := filepath.Join(root, "mounts")
	err := os.MkdirAll(mountPath, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make qsfs mounts dir")
	}

	return &QSFS{
		cl:         cl,
		mountsPath: mountPath,
	}, nil
}

func (q *QSFS) Mount(wlID string, cfg zos.QuatumSafeFS) (mountPath string, err error) {
	defer func() {
		if err != nil {
			if err := q.Unmount(wlID); err != nil {
				log.Error().Err(err).Msg("error cleaning up after qsfs setup failure")
			}
		}
	}()
	networkd := stubs.NewNetworkerStub(q.cl)
	flistd := stubs.NewFlisterStub(q.cl)
	contd := stubs.NewContainerModuleStub(q.cl)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	netns, err := networkd.QSFSPrepare(ctx, wlID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare qsfs")
	}
	flistPath, err := flistd.Mount(ctx, wlID, qsfsFlist, pkg.MountOptions{
		ReadOnly: false,
		Limit:    cfg.Cache,
		Storage:  "zdb://hub.grid.tf:9900",
	})
	if err != nil {
		err = errors.Wrap(err, "failed to mount qsfs flist")
		return
	}
	if lerr := q.writeQSFSConfig(flistPath, cfg.Config); lerr != nil {
		err = errors.Wrap(lerr, "couldn't write qsfs config")
		return
	}
	mountPath = q.mountPath(wlID)
	err = q.prepareMountPath(mountPath)
	if err != nil {
		err = errors.Wrap(err, "failed to prepare mount path")
		return
	}
	cont := pkg.Container{
		Name:        wlID,
		RootFS:      flistPath,
		Entrypoint:  "/sbin/zinit init",
		Interactive: false,
		Network:     pkg.NetworkInfo{Namespace: netns},
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
	if lerr := q.waitUntilMounted(mountPath, 2*time.Minute); lerr != nil {
		err = lerr
		return
	}
	return
}

func (f *QSFS) waitUntilMounted(path string, timeout time.Duration) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(1 * time.Second) {
		mounted, err := f.isMounted(path)
		if err != nil {
			return errors.Wrap(err, "failed to check if zdbfs is mounted")
		}
		if mounted {
			return nil
		}
	}
	return fmt.Errorf("waiting for zdbfs mount %s timedout", path)

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

func (q *QSFS) Unmount(wlID string) error {
	networkd := stubs.NewNetworkerStub(q.cl)
	flistd := stubs.NewFlisterStub(q.cl)
	contd := stubs.NewContainerModuleStub(q.cl)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	// listing all containers and matching the name looks like a lot of work
	if err := contd.Delete(ctx, qsfsContainerNS, pkg.ContainerID(wlID)); err != nil {
		log.Error().Err(err).Msg("failed to delete qsfs container")
	}
	mountPath := q.mountPath(wlID)
	// unmount twice, once for the zdbfs and the self-mount
	if err := syscall.Unmount(mountPath, 0); err != nil {
		log.Error().Err(err).Msg("failed to unmount mount path 1st time")
	}
	if err := syscall.Unmount(mountPath, 0); err != nil {
		log.Error().Err(err).Msg("failed to unmount mount path 2nd time")
	}
	if err := os.RemoveAll(mountPath); err != nil {
		log.Error().Err(err).Msg("failed to remove mountpath dir")
	}
	if err := flistd.Unmount(ctx, wlID); err != nil {
		log.Error().Err(err).Msg("failed to unmount flist")
	}

	if err := networkd.QSFSDestroy(ctx, wlID); err != nil {
		log.Error().Err(err).Msg("failed to destrpy qsfs network")
	}
	return nil
}

func (q *QSFS) mountPath(wlID string) string {
	return filepath.Join(q.mountsPath, wlID)
}

func (q *QSFS) prepareMountPath(path string) error {
	if err := os.Mkdir(path, 0644); err != nil {
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

func (q *QSFS) writeQSFSConfig(root string, cfg zos.QuantumSafeFSConfig) error {
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
