package noded

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
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/geoip"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	reportUptimeEvery = 2 * time.Hour

	tcUrl  = ""
	tcHash = ""
)

func registration(ctx context.Context, cl zbus.Client, env environment.Environment, cap gridtypes.Capacity) (nodeID, twinID uint32, err error) {
	var (
		netMgr = stubs.NewNetworkerStub(cl)
	)

	// we need to collect all node information here
	// - we already have capacity
	// - we get the location (will not change after initial registration)
	loc, err := geoip.Fetch()
	if err != nil {
		log.Fatal().Err(err).Msg("fetch location")
	}

	// - yggdrasil
	// node always register with ndmz address
	var ygg net.IP
	if ips, _, err := netMgr.Addrs(ctx, yggdrasil.YggNSInf, "ndmz"); err == nil {
		if len(ips) == 0 {
			return 0, 0, errors.Wrap(err, "failed to get yggdrasil ip")
		}
		if len(ips) == 1 {
			ygg = net.IP(ips[0])
		}
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
		nodeID, twinID, err = registerNode(ctx, env, cl, sub, cap, loc, ygg)
		return err
	}, bo, retryNotify)

	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to register node")
	}

	// well the node is registed. but now we need to monitor changes to networking
	// to update the node (currently only yggdrasil ip) which will require changes
	// to twin ip.
	go func() {
		for {
			err := watch(ctx, env, cl, sub, cap, loc, ygg)
			if errors.Is(err, context.Canceled) {
				return
			} else if err != nil {
				log.Error().Err(err).Msg("watching network changes failed")
				<-time.After(3 * time.Second)
			}
		}
	}()

	return nodeID, twinID, nil
}

func watch(
	ctx context.Context,
	env environment.Environment,
	cl zbus.Client,
	sub *substrate.Substrate,
	cap gridtypes.Capacity,
	loc geoip.Location,
	ygg net.IP,
) error {
	var (
		netMgr = stubs.NewNetworkerStub(cl)
	)

	yggCh, err := netMgr.YggAddresses(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to register on ygg ips changes")
	}

	log.Info().Msg("start watching node network changes")
	for {
		update := false
		select {
		case <-ctx.Done():
			return ctx.Err()
		case yggInput := <-yggCh:
			var yggNew net.IP
			if len(yggInput) > 0 {
				yggNew = yggInput[0].IP
			}
			if !yggNew.Equal(ygg) {
				ygg = yggNew
				update = true
			}
		}

		if !update {
			continue
		}
		// some of the node config has changed. we need to try register it again
		log.Debug().Msg("node setup seems to have been changed. re-register")
		exp := backoff.NewExponentialBackOff()
		exp.MaxInterval = 2 * time.Minute
		bo := backoff.WithContext(exp, ctx)
		err = backoff.RetryNotify(func() error {
			_, _, err := registerNode(ctx, env, cl, sub, cap, loc, ygg)
			return err
		}, bo, retryNotify)

		if err != nil {
			return errors.Wrap(err, "failed to register node")
		}
	}
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
	ygg net.IP,
) (nodeID, twinID uint32, err error) {
	var (
		mgr    = stubs.NewIdentityManagerStub(cl)
		netMgr = stubs.NewNetworkerStub(cl)
	)

	zosIps, zosMac, err := netMgr.Addrs(ctx, "zos", "")
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to get zos bridge information")
	}

	interfaces := []substrate.Interface{
		{
			Name: "zos",
			Mac:  zosMac,
			IPs: func() []string {
				var ips []string
				for _, ip := range zosIps {
					ipV := net.IP(ip)
					ips = append(ips, ipV.String())
				}
				return ips
			}(),
		},
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
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return 0, 0, err
	}
	if _, err := sub.EnsureAccount(id, env.ActivationURL, tcUrl, tcHash); err != nil {
		return 0, 0, errors.Wrap(err, "failed to ensure account")
	}

	twinID, err = ensureTwin(sub, sk, ygg)
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

		if reflect.DeepEqual(node.Resources, resources) &&
			reflect.DeepEqual(node.Location, location) &&
			reflect.DeepEqual(node.Interfaces, interfaces) &&
			node.Country == loc.Country {
			// so node exists AND pub config, nor resources hasn't changed
			log.Debug().Msg("node information has not changed")
			return uint32(node.ID), uint32(node.TwinID), nil
		}

		// we need to update the node
		node.ID = types.U32(nodeID)
		node.FarmID = types.U32(env.FarmerID)
		node.TwinID = types.U32(twinID)
		node.Resources = resources
		node.Location = location
		node.Country = loc.Country
		node.City = loc.City
		node.Interfaces = interfaces

		log.Debug().Msgf("node data have changing, issuing an update node: %+v", node)
		_, err = sub.UpdateNode(id, *node)
		return uint32(node.ID), uint32(node.TwinID), err
	}

	// create node
	nodeID, err = sub.CreateNode(id, substrate.Node{
		FarmID:     types.U32(env.FarmerID),
		TwinID:     types.U32(twinID),
		Resources:  resources,
		Location:   location,
		Country:    loc.Country,
		City:       loc.City,
		Interfaces: interfaces,
	})

	if err != nil {
		return nodeID, 0, err
	}

	return nodeID, twinID, nil
}

func ensureTwin(sub *substrate.Substrate, sk ed25519.PrivateKey, ip net.IP) (uint32, error) {
	identity, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return 0, err
	}
	twinID, err := sub.GetTwinByPubKey(identity.PublicKey())
	if errors.Is(err, substrate.ErrNotFound) {
		return sub.CreateTwin(identity, ip)
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
	return sub.UpdateTwin(identity, ip)
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
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return err
	}

	for {
		uptime, err := host.Uptime()
		if err != nil {
			return errors.Wrap(err, "failed to get uptime")
		}
		log.Debug().Msg("updating node uptime")
		hash, err := sub.UpdateNodeUptime(id, uptime)
		if err != nil {
			return errors.Wrap(err, "failed to report uptime")
		}

		log.Info().Str("hash", hash.Hex()).Msg("node uptime hash")

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(reportUptimeEvery):
			continue
		}
	}
}
