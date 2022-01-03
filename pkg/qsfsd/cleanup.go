package qsfsd

import (
	"context"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/containerd/containerd/errdefs"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	tombstoneFile = ".death.scheduled"
	UploadLimit   = 100 * 1024 * 1024 // kill it for less 100 MB every 10 minutes
	checkPeriod   = 10 * time.Minute
)

func (q *QSFS) periodicCleanup(ctx context.Context) {
	lastUploadMap := make(map[pkg.ContainerID]uint64)
	t := time.NewTicker(checkPeriod)
	for {
		select {
		case <-t.C:
			if err := q.checkDeadQSFSs(ctx, lastUploadMap); err != nil {
				log.Error().Err(err).Msg("a failed qsfs cleanup round")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (q *QSFS) checkDeadQSFSs(ctx context.Context, lastUploadMap map[pkg.ContainerID]uint64) error {
	contd := stubs.NewContainerModuleStub(q.cl)
	containers, err := contd.List(ctx, qsfsContainerNS)
	if err != nil {
		return errors.Wrap(err, "couldn't list qsfs containers")
	}
	for _, contID := range containers {
		marked, err := q.isMarkedForDeletion(ctx, string(contID))
		if err != nil {
			log.Err(err).Msg("mark check failed")
			continue
		}
		if !marked {
			continue
		}
		metrics, err := q.qsfsMetrics(ctx, string(contID))
		if err != nil {
			log.Err(err).Str("id", string(contID)).Msg("couldn't get qsfs metrics")
			continue
		}
		uploaded := metrics.NetTxBytes
		if lastUploaded, ok := lastUploadMap[contID]; ok && uploaded-lastUploaded < UploadLimit {
			// didn't upload enough => dead
			q.Unmount(string(contID))
			delete(lastUploadMap, contID)
		} else {
			// first time or uploaded a lot in the last 10 minutes
			lastUploadMap[contID] = uploaded
		}
	}
	return nil
}
func (q *QSFS) isMarkedForDeletion(ctx context.Context, wlID string) (bool, error) {
	contd := stubs.NewContainerModuleStub(q.cl)
	contID := pkg.ContainerID(wlID)
	cont, err := contd.Inspect(ctx, qsfsContainerNS, contID)

	if errors.Is(err, errdefs.ErrNotFound) {
		// not found
		//
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "failed to fetch qsfs container for a cleanup check")
	}
	tombstonePath := tombstone(cont.RootFS)
	_, err = os.Stat(tombstonePath)
	if errors.Is(err, os.ErrNotExist) {
		// not dead
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to check the container death mark")
	}
	return true, nil
}
func (q *QSFS) markDelete(ctx context.Context, wlID string) error {
	contd := stubs.NewContainerModuleStub(q.cl)
	cont, err := contd.Inspect(ctx, qsfsContainerNS, pkg.ContainerID(wlID))
	if err != nil {
		return errors.Wrap(err, "couldn't get the qsfs container")
	}
	tombstonePath := tombstone(cont.RootFS)
	file, err := os.Create(tombstonePath)
	if err != nil {
		return errors.Wrap(err, "couldn't mark qsfs container for deletion")
	}
	file.Close()
	return nil
}

func tombstone(rootfs string) string {
	return path.Join(rootfs, tombstoneFile)
}

func (q *QSFS) Unmount(wlID string) {
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
	for i := 0; i < 2; i++ {
		if err := syscall.Unmount(mountPath, 0); err != nil && !errors.Is(err, syscall.EINVAL) {
			log.Error().Err(err).Msg("failed to unmount mount path 1st time")
		}
	}
	if err := os.RemoveAll(mountPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("failed to remove mountpath dir")
	}
	if err := flistd.Unmount(ctx, wlID); err != nil {
		log.Error().Err(err).Msg("failed to unmount flist")
	}

	if err := networkd.QSFSDestroy(ctx, wlID); err != nil {
		if _, ok := err.(ns.NSPathNotExistErr); !ok {
			// log any error other than that the namespace doesn't exist
			log.Error().Err(err).Msg("failed to destrpy qsfs network")
		}
	}
}
