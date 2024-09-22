package qsfsd

import (
	"context"
	"os"
	"path"
	"path/filepath"
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
	UploadLimit = 100 * 1024 * 1024 // kill it for less 100 MB every 10 minutes
	checkPeriod = 10 * time.Minute
)

type failedQSFSState struct {
	lastUploadMap       map[string]uint64
	metricsFailureCount map[string]uint
}

func newState() *failedQSFSState {
	return &failedQSFSState{
		lastUploadMap:       make(map[string]uint64),
		metricsFailureCount: make(map[string]uint),
	}
}
func (f *failedQSFSState) delete(wlID string) {
	delete(f.lastUploadMap, wlID)
	delete(f.metricsFailureCount, wlID)
}

func (q *QSFS) periodicCleanup(ctx context.Context) {
	state := newState()
	t := time.NewTicker(checkPeriod)
	for {
		select {
		case <-t.C:
			if err := q.checkDeadQSFSs(ctx, state); err != nil {
				log.Error().Err(err).Msg("a failed qsfs cleanup round")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (q *QSFS) checkDeadQSFSs(ctx context.Context, state *failedQSFSState) error {
	paths, err := filepath.Glob(filepath.Join(q.tombstonesPath, "*"))
	if err != nil {
		return errors.Wrap(err, "couldn't list deleted containers")
	}
	for _, path := range paths {
		wlID := filepath.Base(path)
		metrics, err := q.qsfsMetrics(ctx, wlID)
		if err != nil {
			log.Err(err).Str("id", wlID).Msg("couldn't get qsfs metrics")
			state.metricsFailureCount[wlID] += 1
			if state.metricsFailureCount[wlID] >= 10 {
				q.Unmount(wlID)
				state.delete(wlID)
			}
			continue
		}
		uploaded := metrics.NetTxBytes
		if lastUploaded, ok := state.lastUploadMap[wlID]; ok && uploaded-lastUploaded < UploadLimit {
			// didn't upload enough => dead
			q.Unmount(wlID)
			state.delete(wlID)
		} else {
			// first time or uploaded a lot in the last 10 minutes
			state.lastUploadMap[wlID] = uploaded
		}
	}
	return nil
}

func (q *QSFS) isMarkedForDeletion(ctx context.Context, wlID string) (bool, error) {
	tombstonePath := q.tombstone(wlID)
	_, err := os.Stat(tombstonePath)
	if errors.Is(err, os.ErrNotExist) {
		// not dead
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to check the container death mark")
	}
	return true, nil
}

func (q *QSFS) isOldMarkedForDeletion(ctx context.Context, wlID string) (bool, error) {
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
	tombstonePath := filepath.Join(cont.RootFS, ".death.scheduled")
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
	tombstonePath := q.tombstone(wlID)
	file, err := os.Create(tombstonePath)
	if err != nil {
		return errors.Wrap(err, "couldn't mark qsfs container for deletion")
	}
	file.Close()
	return nil
}

func (q *QSFS) tombstone(wlID string) string {
	return path.Join(q.tombstonesPath, wlID)
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
	if err := os.RemoveAll(q.tombstone(wlID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("failed to remove tombstone path")
	}
	if err := flistd.Unmount(ctx, wlID); err != nil {
		log.Error().Err(err).Msg("failed to unmount flist")
	}

	if err := networkd.QSFSDestroy(ctx, wlID); err != nil {
		if _, ok := err.(ns.NSPathNotExistErr); !ok {
			// log any error other than that the namespace doesn't exist
			log.Error().Err(err).Msg("failed to destroy qsfs network")
		}
	}
}
