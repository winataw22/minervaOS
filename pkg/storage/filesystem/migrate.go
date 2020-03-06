package filesystem

import (
	"context"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	zosV1PoolPrefix = "sp_"
)

func wipe(ctx context.Context, device *Device, exe executer) error {
	_, err := exe.run(ctx, "wipefs", "-a", "-f", device.Path)
	return err
}

// Migrate is a simple migration routine that makes sure we have
// fresh disks to use from zos v1, in case you moving
// away from older version
// After migration you should alway use the returned Device manager, or create
// a new one.
func Migrate(ctx context.Context, m DeviceManager) (DeviceManager, error) {
	return migrate(ctx, m, executerFunc(run))
}

func migrate(ctx context.Context, m DeviceManager, exe executer) (DeviceManager, error) {
	devices, err := m.Raw(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "migration failed, couldn't list disks")
	}

	var destroyMe []Device
loop:
	for _, device := range devices {
		if strings.HasPrefix(device.Label, zosV1PoolPrefix) {
			destroyMe = append(destroyMe, device)
			continue
		}

		for _, child := range device.Children {
			if strings.HasPrefix(child.Label, zosV1PoolPrefix) {
				destroyMe = append(destroyMe, device)
				continue loop
			}
		}
	}

	for _, device := range destroyMe {
		log.Warn().Str("device", device.Path).Msg("wiping device")
		if err := wipe(ctx, &device, exe); err != nil {
			log.Error().Err(err).Str("device", device.Path).Msg("failed to wipe disk")
		}
	}

	if len(destroyMe) > 0 {
		syscall.Sync()
		if out, err := exe.run(ctx, "partprobe"); err != nil {
			log.Error().Err(err).Str("out", string(out)).Msg("failed to partprobe")
		}
	}

	return m.Reset(), nil
}
