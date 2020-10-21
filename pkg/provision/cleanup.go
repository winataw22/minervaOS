package provision

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/provision/common"
	"github.com/threefoldtech/zos/pkg/storage"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zdb"
	"golang.org/x/net/context"
)

// CleanupResources cleans up unused resources
func CleanupResources(ctx context.Context, zbus zbus.Client) error {
	explorer, err := app.ExplorerClient()
	if err != nil {
		return err
	}
	storaged := stubs.NewStorageModuleStub(zbus)

	toSave, err := checkContainers(ctx, zbus)
	if err != nil {
		return errors.Wrap(err, "failed to check containers")
	}

	// ListFilesystems do not return the special cache and vdisk filesystem
	// so we are safe to process everything that is returned
	fss, err := storaged.ListFilesystems()
	if err != nil {
		return err
	}

	for _, fs := range fss {
		log.Info().Msgf("checking subvol %s", fs.Path)

		// Now, is this subvol in one of the toSave ?
		if _, ok := toSave[fs.Path]; ok {
			log.Info().Msgf("skipping volume '%s' is used", fs.Path)
			continue
		}

		if len(fs.Name) == 64 {
			// if the fs is not used by any container and its name is 64 character long
			// they are left over of old containers when flistd used to generate random names
			// for the container root flist subvolumes
			log.Info().Msgf("delete root container flist subvolume '%s'", fs.Path)
			if err := storaged.ReleaseFilesystem(fs.Name); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", fs.Path)
			}
			continue
		}

		if strings.HasPrefix(fs.Name, storage.ZDBPoolPrefix) {
			// we can safely delete this one because it is not used by any container
			// this is ensured line 46
			log.Info().Msgf("delete left over 0-DB subvolume '%s'", fs.Path)
			if err := storaged.ReleaseFilesystem(fs.Name); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", fs.Path)
			}
			continue
		}

		if fs.Name == "fcvms" {
			// left over from testing during vm module development
			log.Info().Msgf("delete fcvm subvolume '%s'", fs.Path)
			if err := storaged.ReleaseFilesystem(fs.Name); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", fs.Path)
			}
			continue
		}

		// Is this subvol not in toSave?
		// Check the explorer if it needs to be deleted
		delete := checkReservationToDelete(fs, explorer)
		if delete {
			log.Info().Msgf("deleting subvolume %s", fs.Path)
			if err := storaged.ReleaseFilesystem(fs.Name); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", fs.Path)
			}
			continue
		}
		log.Info().Msgf("skipping subvolume %s", fs.Path)
	}

	return nil
}

func checkReservationToDelete(fs pkg.Filesystem, cl *client.Client) bool {

	wid, err := WorkloadIDFromFilesystem(fs)
	if err != nil {
		log.Err(err).Msgf("failed to convert workload id %d", wid)
		return false
	}

	log.Info().Msgf("checking explorer for reservation: %d", wid)
	reservation, err := cl.Workloads.Get(schema.ID(wid))
	if err != nil {
		var hErr client.HTTPError
		if ok := errors.As(err, &hErr); ok {
			resp := hErr.Response()
			// If reservation is not found it should be deleted
			if resp.StatusCode == 404 {
				return true
			}
		}
		return false
	}

	if reservation.GetNextAction() == workloads.NextActionDelete {
		log.Info().Msgf("workload %d has next action to delete", wid)
		return true
	}

	return false
}

// checks running containers for subvolumes that might need to be saved because they are used
// and subvolumes that might need to be deleted because they have no attached container anymore
func checkContainers(ctx context.Context, zbus zbus.Client) (map[string]struct{}, error) {
	toSave := make(map[string]struct{})

	contd := stubs.NewContainerModuleStub(zbus)

	cNamespaces, err := contd.ListNS()
	if err != nil {
		log.Err(err).Msgf("failed to list namespaces")
		return nil, err
	}

	for _, ns := range cNamespaces {
		containerIDs, err := contd.List(ns)
		if err != nil {
			log.Error().Err(err).Msg("failed to list container IDs")
			return nil, err
		}

		for _, id := range containerIDs {
			ctr, err := contd.Inspect(ns, id)
			if err != nil {
				log.Error().Err(err).Msgf("failed to inspect container %s", id)
				return nil, err
			}

			log.Info().Msgf("container ID %s", id)
			var zdbNamespaces []string
			if ns == "zdb" {
				zdbNamespaces, err = listNamespaces(string(id))
				if err != nil {
					log.Err(err).Msg("failed to list container namespaces")
					continue
				}
			}

			// avoid to remove any used subvolume used by flistd for root container fs
			toSave[ctr.RootFS] = struct{}{}

			for _, mnt := range ctr.Mounts {
				// TODO: do we need this check ?
				// if !strings.HasPrefix(mnt.Source, "/mnt/") {
				// 	continue
				// }
				if len(zdbNamespaces) == 1 && zdbNamespaces[0] == "default" {
					err := common.DeleteZdbContainer(id, zbus)
					if err != nil {
						log.Err(err).Msgf("failed to delete zdb container %s", string(id))
						continue
					}
				} else {
					toSave[mnt.Source] = struct{}{}
				}

			}
		}
	}

	return toSave, nil
}

func socketDir(containerID string) string {
	return fmt.Sprintf("/var/run/zdb_%s", containerID)
}

func initZdbConnection(id string) zdb.Client {
	socket := fmt.Sprintf("unix://%s/zdb.sock", socketDir(id))
	return zdb.New(socket)
}

func listNamespaces(containterID string) ([]string, error) {
	zdbCl := initZdbConnection(containterID)
	if err := zdbCl.Connect(); err != nil {
		log.Err(err).Msgf("failed to connect to 0-db: %s", containterID)
		return nil, err
	}

	zdbNamespaces, err := zdbCl.Namespaces()
	if err != nil {
		log.Err(err).Msg("failed to retrieve zdb namespaces")
		return nil, err
	}
	defer zdbCl.Close()

	return zdbNamespaces, nil
}
