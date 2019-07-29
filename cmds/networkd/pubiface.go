package main

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
)

func watchPubIface(ctx context.Context, nodeID modules.Identifier, db network.TNoDB, ifaceVersion int) <-chan *network.PubIface {
	var currentVersion = ifaceVersion

	ch := make(chan *network.PubIface)
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

			exitIface, err := db.ReadPubIface(nodeID)
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

func configuePubIface(iface *network.PubIface) error {
	cleanup := func() error {
		pubNs, err := namespace.GetByName(network.PublicNamespace)
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
