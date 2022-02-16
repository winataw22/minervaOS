package storage

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	zdbVolume = "zdb"
)

//Devices list all "allocated" devices
func (m *Module) Devices() ([]pkg.Device, error) {
	var devices []pkg.Device
	log.Debug().Int("disks", len(m.hdds)).Msg("listing zdb devices")
	for _, hdd := range m.hdds {
		log.Debug().Str("device", hdd.Path()).Msg("checking device")
		if _, err := hdd.Mounted(); err != nil {
			log.Debug().Str("device", hdd.Path()).Msg("not mounted")
			continue
		}

		volumes, err := hdd.Volumes()
		if err != nil {
			log.Error().Err(err).Str("pool", hdd.Name()).Msg("failed to get pool volumes")
			continue
		}
		usage, err := hdd.Usage()
		if err != nil {
			return nil, err
		}
		for _, vol := range volumes {
			log.Debug().Str("volume", vol.Path()).Str("name", vol.Name()).Msg("checking volume")
			if vol.Name() != zdbVolume {
				continue
			}

			devices = append(devices, pkg.Device{
				ID:   hdd.Name(),
				Path: vol.Path(),
				Usage: pkg.Usage{
					Size: gridtypes.Unit(usage.Size),
					Used: gridtypes.Unit(usage.Used),
				},
			})
			break
		}
	}

	return devices, nil
}

// DeviceLookup looks up device by name
func (m *Module) DeviceLookup(name string) (pkg.Device, error) {
	for _, hdd := range m.hdds {
		if hdd.Name() != name {
			continue
		}

		if _, err := hdd.Mounted(); err != nil {
			return pkg.Device{}, errors.Wrap(err, "device is not allocated")
		}

		volumes, err := hdd.Volumes()
		if err != nil {
			return pkg.Device{}, errors.Wrap(err, "failed to get pool volumes")
		}

		usage, err := hdd.Usage()
		if err != nil {
			return pkg.Device{}, err
		}
		for _, vol := range volumes {
			if vol.Name() != zdbVolume {
				continue
			}

			return pkg.Device{
				ID:   hdd.Name(),
				Path: vol.Path(),
				Usage: pkg.Usage{
					Size: gridtypes.Unit(usage.Size),
					Used: gridtypes.Unit(usage.Used),
				},
			}, nil
		}

		return pkg.Device{}, errors.Wrap(err, "device is not allocated (no zdb volume found)")
	}

	return pkg.Device{}, fmt.Errorf("device not found")
}

// DeviceAllocate allocates a new free device, allocation is done
// by creation a zdb subvolume
func (m *Module) DeviceAllocate(min gridtypes.Unit) (pkg.Device, error) {
	for _, hdd := range m.hdds {
		if _, err := hdd.Mounted(); err == nil {
			// mounted pool. skip
			continue
		}

		if hdd.Device().Size < uint64(min) {
			continue
		}

		if _, err := hdd.Mount(); err != nil {
			log.Error().Err(err).Str("pool", hdd.Name()).Msg("failed to mount pool")
			continue
		}

		volumes, err := hdd.Volumes()
		if err != nil {
			log.Error().Err(err).Str("pool", hdd.Name()).Msg("failed to get pool volumes")
			continue
		}

		if len(volumes) != 0 {
			log.Info().Str("pool", hdd.Name()).Msg("pool is already used")
			continue
		}

		volume, err := hdd.AddVolume(zdbVolume)
		if err != nil {
			log.Error().Err(err).Msg("failed to create zdb volume")
			continue
		}

		usage, err := hdd.Usage()
		if err != nil {
			return pkg.Device{}, err
		}

		return pkg.Device{
			ID:   hdd.Name(),
			Path: volume.Path(),
			Usage: pkg.Usage{
				Size: gridtypes.Unit(usage.Size),
				Used: gridtypes.Unit(usage.Used),
			},
		}, nil

	}

	return pkg.Device{}, fmt.Errorf("no more free devices found")
}
