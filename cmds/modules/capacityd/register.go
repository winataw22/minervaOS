package capacityd

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/geoip"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/substrate"
)

const (
	reportUptimeEvery = 8 * time.Hour
)

func registration(ctx context.Context, cl zbus.Client, cap gridtypes.Capacity) (nodeID, twinID uint32, err error) {
	env, err := environment.Get()
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to get runtime environment for zos")
	}

	loc, err := geoip.Fetch()
	if err != nil {
		log.Fatal().Err(err).Msg("fetch location")
	}

	log.Debug().
		Uint64("cru", cap.CRU).
		Uint64("mru", uint64(cap.MRU)).
		Uint64("sru", uint64(cap.SRU)).
		Uint64("hru", uint64(cap.HRU)).
		Msg("node capacity")

	sub, err := env.GetSubstrate()
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to create substrate client")
	}

	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 2 * time.Minute
	bo := backoff.WithContext(exp, ctx)
	err = backoff.RetryNotify(func() error {
		nodeID, twinID, err = registerNode(ctx, env, cl, sub, cap, loc)
		return err
	}, bo, retryNotify)

	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to register node")
	}

	return nodeID, twinID, nil
}

func retryNotify(err error, d time.Duration) {
	log.Warn().Err(err).Str("sleep", d.String()).Msg("registration failed")
}

func registerNode(
	ctx context.Context,
	env environment.Environment,
	cl zbus.Client,
	sub *substrate.Substrate,
	cap gridtypes.Capacity,
	loc geoip.Location,
) (nodeID, twinID uint32, err error) {
	var (
		mgr    = stubs.NewIdentityManagerStub(cl)
		netMgr = stubs.NewNetworkerStub(cl)
	)

	var pubCfg substrate.OptionPublicConfig
	if pub, err := netMgr.GetPublicConfig(ctx); err == nil {
		pubCfg.HasValue = true
		pubCfg.AsValue = substrate.PublicConfig{
			IPv4: pub.IPv4.String(),
			GWv4: pub.GW4.String(),
			IPv6: pub.IPv6.String(),
			GWv6: pub.GW6.String(),
		}
	}

	resources := substrate.Resources{
		HRU: types.U64(cap.HRU),
		SRU: types.U64(cap.SRU),
		CRU: types.U64(cap.CRU),
		MRU: types.U64(cap.MRU),
	}

	location := substrate.Location{
		Longitude: fmt.Sprint(loc.Longitute),
		Latitude:  fmt.Sprint(loc.Latitude),
	}

	log.Info().Str("id", mgr.NodeID(ctx).Identity()).Msg("start registration of the node")
	log.Info().Msg("registering node on blockchain")

	sk := ed25519.PrivateKey(mgr.PrivateKey(ctx))
	id, err := substrate.IdentityFromSecureKey(sk)
	if err != nil {
		return 0, 0, err
	}
	if _, err := sub.EnsureAccount(&id); err != nil {
		return 0, 0, errors.Wrap(err, "failed to ensure account")
	}

	// make sure the node twin exists
	cfg := yggdrasil.GenerateConfig(sk)
	address, err := cfg.Address()
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to get yggdrasil address")
	}

	twinID, err = ensureTwin(sub, sk, address)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to ensure twin")
	}

	nodeID, err = sub.GetNodeByTwinID(twinID)
	if err != nil && !errors.Is(err, substrate.ErrNotFound) {
		return 0, 0, err
	} else if err == nil {
		// node exists. we validate everything is good
		// otherwise we update the node
		log.Debug().Uint32("node", nodeID).Msg("node already found on blockchain")
		node, err := sub.GetNode(nodeID)
		if err != nil {
			return 0, 0, errors.Wrapf(err, "failed to get node with id: %d", nodeID)
		}

		if reflect.DeepEqual(node.PublicConfig, pubCfg) &&
			reflect.DeepEqual(node.Resources, resources) &&
			reflect.DeepEqual(node.Location, location) {
			// so node exists AND pub config, nor resources hasn't changed
			log.Debug().Msg("node information has not changed")
			return uint32(node.ID), uint32(node.TwinID), nil
		}

		// we need to update the node
		node.PublicConfig = pubCfg
		node.Resources = resources
		node.Location = location

		log.Debug().Msg("node data have changing, issuing an update node")
		_, err = sub.UpdateNode(&id, *node)
		return uint32(node.ID), uint32(node.TwinID), err
	}

	// create node
	nodeID, err = sub.CreateNode(&id, substrate.Node{
		FarmID:       types.U32(env.FarmerID),
		TwinID:       types.U32(twinID),
		Resources:    resources,
		Location:     location,
		Country:      loc.CountryCode,
		City:         loc.City,
		PublicConfig: pubCfg,
	})

	if err != nil {
		return nodeID, 0, err
	}

	return nodeID, twinID, nil
}

func ensureTwin(sub *substrate.Substrate, sk ed25519.PrivateKey, ip net.IP) (uint32, error) {
	identity, err := substrate.IdentityFromSecureKey(sk)
	if err != nil {
		return 0, err
	}
	twinID, err := sub.GetTwinByPubKey(identity.PublicKey)
	if errors.Is(err, substrate.ErrNotFound) {
		return sub.CreateTwin(&identity, ip)
	} else if err != nil {
		return 0, errors.Wrap(err, "failed to list twins")
	}

	twin, err := sub.GetTwin(twinID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get twin object")
	}

	if twin.IP == ip.String() {
		return twinID, nil
	}

	// update twin to new ip
	return sub.UpdateTwin(&identity, ip)
}

func uptime(ctx context.Context, cl zbus.Client) error {
	var (
		mgr = stubs.NewIdentityManagerStub(cl)
	)
	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get runtime environment for zos")
	}

	sub, err := env.GetSubstrate()
	if err != nil {
		return errors.Wrap(err, "failed to create substrate client")
	}

	sk := ed25519.PrivateKey(mgr.PrivateKey(ctx))
	id, err := substrate.IdentityFromSecureKey(sk)
	if err != nil {
		return err
	}
	first := make(chan struct{}, 1)
	first <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-first:
		case <-time.After(reportUptimeEvery):
		}
		uptime, err := host.Uptime()
		if err != nil {
			return errors.Wrap(err, "failed to get uptime")
		}
		log.Debug().Msg("updating node uptime")
		if err := sub.UpdateNodeUptime(&id, uptime); err != nil {
			log.Error().Err(err).Msg("failed to report uptime")
		}
	}
}
