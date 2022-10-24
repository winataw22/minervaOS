package provisiond

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision/storage"
	fsStorage "github.com/threefoldtech/zos/pkg/provision/storage.fs"
)

func storageMigration(db *storage.BoltStorage, fs *fsStorage.Fs) error {
	log.Info().Msg("starting storage migration")
	twins, err := fs.Twins()
	if err != nil {
		return err
	}
	migration := db.Migration()
	errorred := false
	for _, twin := range twins {
		dls, err := fs.ByTwin(twin)
		if err != nil {
			log.Error().Err(err).Uint32("twin", twin).Msg("failed to list twin deployments")
			continue
		}

		sort.Slice(dls, func(i, j int) bool {
			return dls[i] < dls[j]
		})

		for _, dl := range dls {
			log.Info().Uint32("twin", twin).Uint64("deployment", dl).Msg("processing deployment migration")
			deployment, err := fs.Get(twin, dl)
			if err != nil {
				log.Error().Err(err).Uint32("twin", twin).Uint64("deployment", dl).Msg("failed to get deployment")
				errorred = true
				continue
			}
			if err := migration.Migrate(deployment); err != nil {
				log.Error().Err(err).Uint32("twin", twin).Uint64("deployment", dl).Msg("failed to migrate deployment")
				errorred = true
				continue
			}
			if err := fs.Delete(deployment); err != nil {
				log.Error().Err(err).Uint32("twin", twin).Uint64("deployment", dl).Msg("failed to delete migrated deployment")
				continue
			}
		}
	}

	if errorred {
		return fmt.Errorf("not all deployments where migrated")
	}

	return nil
}

func netResourceMigration(active []gridtypes.Deployment) error {
	/*
		because of limit on the net devices names (length mainly) it was always needed to
		name the devices with unique name that is derived from the actual user twin/deployment and network workload name
		hence the zos.NetworkID function which takes into account all required inputs to make a unique network id.

		The problem now it's impossible for the system to map back network resources names to a unique reservation.
		a bridge br-27xVrq9bva3vJ or a namespace n-27xVrq9bva3vJ means nothing and you can't tell which user owns this.

		Since networkd stores the network object anyway on disk (under /var/run/cache/networkd/networks) it's then possible to update those objects
		to also contain the workload full id not only the network id.

		The first way to do this is to update all cached files on this volatile storage, update the file version and add the extra missing filed. but this
		requires changes in multiple places (define a new type and track the version of the file). make sure the correct types are used and possibly support
		of multiple versions of the structure in multiple places.

		The other "easier" approach is simply creating a symlink from the ID to the correct network file. this means only entities that need to find the network
		by it's full workload name can use the link to find the NR bridge and namespace.

		Networkd will be modified to always create the symlink (if not exist already) to the persisted NR file. But for already created networks, it's not possible to
		create them from within networkd because it does not know this information.

		Hence this migration code witch will go over all active deployments on start, and create the missing symlinks. Newer networks objects will be created with
		their proper symlinks by networkd.
	*/

	const volatile = "/var/run/cache/networkd/networks"
	_, err := os.Stat(volatile)
	if os.IsNotExist(err) {
		// if this doesn't exist it means it's probably first start (after boot) and hence networkd
		// will be called to create all NRs hence it will create the symlink and nothing we need to
		// do now.
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to check networkd volatile cache")
	}

	sym := filepath.Join(volatile, "link")
	if err := os.MkdirAll(sym, 0755); err != nil {
		return errors.Wrap(err, "failed to create network link directory")
	}

	for _, dl := range active {
		for _, wl := range dl.Workloads {
			if wl.Type != zos.NetworkType ||
				!wl.Result.State.IsOkay() {
				continue
			}
			id, err := gridtypes.NewWorkloadID(dl.TwinID, dl.ContractID, wl.Name)
			if err != nil {
				log.Error().Err(err).Msg("failed to build network workload id")
				continue
			}

			netId := zos.NetworkID(dl.TwinID, wl.Name)
			if _, err := os.Stat(filepath.Join(volatile, netId.String())); os.IsNotExist(err) {
				continue
			}

			if err := os.Symlink(
				filepath.Join("..", string(netId)),
				filepath.Join(sym, string(id)),
			); err != nil && !os.IsExist(err) {
				log.Error().Err(err).Msgf("failed to create network symlink for %s -> ../%s", id, netId)
			}
		}
	}

	return nil
}
