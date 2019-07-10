package main

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
)

func watchPubIface(ctx context.Context, db network.TNoDB) <-chan *network.ExitIface {
	var (
		currentVersion = -1
		err            error
		nodeID         identity.Identifier
	)

	ch := make(chan *network.ExitIface)
	go func() {
		defer func() {
			close(ch)
		}()

		for {
			<-time.After(10 * time.Second)

			log.Info().Msg("check if a public interface if configured for us")

			if nodeID == nil {
				nodeID, err = identity.LocalNodeID()
				if err != nil {
					log.Error().Err(err).Msg("failed to get local node ID")
					continue
				}
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
				log.Info().
					Int("current version", currentVersion).
					Int("received version", exitIface.Version).
					Msg("public interface already configured")
				continue
			}
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

func configuePubIface(iface *network.ExitIface) error {
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
		log.Error().Err(err).Msg("failed to configure public namespace")
		_ = cleanup()
		return err
	}

	return nil
}
