package main

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/types"
)

func watchPubIface(ctx context.Context, nodeID pkg.Identifier, db network.TNoDB, ifaceVersion int) <-chan *types.PubIface {
	var currentVersion = ifaceVersion

	ch := make(chan *types.PubIface)
	go func() {
		defer func() {
			close(ch)
		}()

		for {
			select {
			case <-time.After(time.Second * 10):
			case <-ctx.Done():
				break
			}

			exitIface, err := db.GetPubIface(nodeID)
			if err != nil {
				if err == network.ErrNoPubIface {
					continue
				}
				log.Error().Err(err).Msg("failed to read public interface")
				continue
			}

			if exitIface.Version <= currentVersion {
				continue
			}
			log.Info().
				Int("current version", currentVersion).
				Int("received version", exitIface.Version).
				Msg("new version of the public interface configuration")
			currentVersion = exitIface.Version

			select {
			case ch <- exitIface:
			case <-ctx.Done():
				break
			}
		}
	}()
	return ch
}

func configurePubIface(iface *types.PubIface) error {
	cleanup := func() error {
		pubNs, err := namespace.GetByName(types.PublicNamespace)
		if err != nil {
			log.Error().Err(err).Msg("failed to find public namespace")
			return err
		}
		if err = namespace.Delete(pubNs); err != nil {
			log.Error().Err(err).Msg("failed to delete public namespace")
			return err
		}
		return nil
	}

	if err := network.CreatePublicNS(iface); err != nil {
		_ = cleanup()
		return errors.Wrap(err, "failed to configure public namespace")
	}

	return nil
}
