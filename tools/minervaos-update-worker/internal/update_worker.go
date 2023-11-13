package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/rs/zerolog/log"
	client "github.com/threefoldtech/substrate-client"
)

type Network string

var (
	MainNetwork Network = "production"
	TestNetwork Network = "testing"
	QANetwork   Network = "qa"
)

type Params struct {
	Interval time.Duration
	QAUrls   []string
	TestUrls []string
	MainUrls []string
}

type Worker struct {
	src string
	dst string

	interval  time.Duration
	substrate map[Network]client.Manager
}

// NewWorker creates a new instance of the worker
func NewWorker(src string, dst string, params Params) (*Worker, error) {

	// we need to recalculate the path of the symlink here because of the following
	// - assume we run the tool like `updater -d dst -s src`
	// - it's then gonna build the links as above.
	// - then it will crease dst/zos:testing-3:latest.flist that points to dst/zos:<v>.flist
	// and that is wrong because now the link points to a wrong path. it instead need to be ../dst/<file>
	// so recalculating here
	// we need to find a abs path from dst to src.
	// so this goes as this
	// - we make sure that src and dst are always abs
	// this later will allow us to calculate relative path from dst to src

	src, err := filepath.Abs(src)
	if err != nil {
		return nil, fmt.Errorf("failed to get src as abs path: %w", err)
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to get dst as abs path: %w", err)
	}

	log.Info().Str("src", src).Str("dst", dst).Msg("paths")

	substrate := map[Network]client.Manager{}

	if len(params.QAUrls) != 0 {
		substrate[QANetwork] = client.NewManager(params.QAUrls...)
	}

	if len(params.TestUrls) != 0 {
		substrate[TestNetwork] = client.NewManager(params.TestUrls...)
	}

	if len(params.MainUrls) != 0 {
		substrate[MainNetwork] = client.NewManager(params.MainUrls...)
	}

	return &Worker{
		src:       src,
		dst:       dst,
		substrate: substrate,
		interval:  params.Interval,
	}, nil
}

// checkNetwork to check if a network is valid against main, qa, test
func checkNetwork(network Network) error {
	if network != MainNetwork && network != QANetwork && network != TestNetwork {
		return fmt.Errorf("invalid network")
	}

	return nil
}

// updateZosVersion updates the latest zos flist for a specific network with the updated zos version
func (w *Worker) updateZosVersion(network Network, manager client.Manager) error {
	if err := checkNetwork(network); err != nil {
		return err
	}

	con, err := manager.Substrate()
	if err != nil {
		return err
	}
	defer con.Close()

	currentZosVersion, err := con.GetZosVersion()
	if err != nil {
		return err
	}

	log.Debug().Msgf("getting substrate version %v for network %v", currentZosVersion, network)

	// now we need to find how dst is relative to src
	path, err := filepath.Rel(w.dst, w.src)
	if err != nil {
		return fmt.Errorf("failed to get dst relative path to src: %w", err)
	}

	zosCurrent := fmt.Sprintf("%v/.tag-%v", w.src, currentZosVersion)
	zosLatest := fmt.Sprintf("%v/%v", w.dst, network)
	// the link is like zosCurrent but it has the path relative from the symlink
	// point of view (so relative to the symlink, how to reach zosCurrent)
	// hence the link is instead used in all calls to symlink
	link := fmt.Sprintf("%v/.tag-%v", path, currentZosVersion)

	// check if current exists
	if _, err := os.Lstat(zosCurrent); err != nil {
		return err
	}

	// check if symlink exists
	dst, err := os.Readlink(zosLatest)

	// if no symlink, then create it
	if os.IsNotExist(err) {
		log.Info().Str("from", zosLatest).Str("to", zosCurrent).Msg("linking")
		return os.Symlink(link, zosLatest)
	} else if err != nil {
		return err
	}

	// check if symlink is valid and exists
	if filepath.Base(dst) == filepath.Base(zosCurrent) {
		log.Debug().Msgf("symlink %v to %v already exists", zosCurrent, zosLatest)
		return nil
	}

	// remove symlink if it is not valid and exists
	if err := os.Remove(zosLatest); err != nil {
		return err
	}

	log.Info().Str("from", zosLatest).Str("to", zosCurrent).Msg("linking")
	return os.Symlink(link, zosLatest)
}

// UpdateWithInterval updates the latest zos flist for a specific network with the updated zos version
// with a specific interval between each update
func (w *Worker) UpdateWithInterval(ctx context.Context) {
	ticker := time.NewTicker(w.interval)

	for {
		for network, manager := range w.substrate {
			log.Debug().Msgf("updating zos version for %v", network)

			exp := backoff.NewExponentialBackOff()
			exp.MaxInterval = 2 * time.Second
			exp.MaxElapsedTime = 10 * time.Second
			err := backoff.Retry(func() error {

				err := w.updateZosVersion(network, manager)
				if err != nil {
					log.Error().Err(err).Msg("update failure. retrying")
				}
				return err

			}, exp)

			if err != nil {
				log.Error().Err(err).Msg("update zos failed with error")
			}
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
