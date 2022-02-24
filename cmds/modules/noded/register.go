package noded

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
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

type RegistrationInfo struct {
	Capacity     gridtypes.Capacity
	Location     geoip.Location
	Ygg          net.IP
	SecureBoot   bool
	Virtualized  bool
	SerialNumber string
}

func (r RegistrationInfo) WithCapacity(v gridtypes.Capacity) RegistrationInfo {
	r.Capacity = v
	return r
}

func (r RegistrationInfo) WithLocation(v geoip.Location) RegistrationInfo {
	r.Location = v
	return r
}

func (r RegistrationInfo) WithYggdrail(v net.IP) RegistrationInfo {
	r.Ygg = v
	return r
}

func (r RegistrationInfo) WithSecureBoot(v bool) RegistrationInfo {
	r.SecureBoot = v
	return r
}

func (r RegistrationInfo) WithVirtualized(v bool) RegistrationInfo {
	r.Virtualized = v
	return r
}

func (r RegistrationInfo) WithSerialNumber(v string) RegistrationInfo {
	r.SerialNumber = v
	return r
}

func registration(ctx context.Context, cl zbus.Client, env environment.Environment, info RegistrationInfo) (nodeID, twinID uint32, err error) {
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
		Uint64("cru", info.Capacity.CRU).
		Uint64("mru", uint64(info.Capacity.MRU)).
		Uint64("sru", uint64(info.Capacity.SRU)).
		Uint64("hru", uint64(info.Capacity.HRU)).
		Msg("node capacity")

	sub, err := environment.GetSubstrate()
	if err != nil {
		return 0, 0, err
	}
	info = info.WithLocation(loc).WithYggdrail(ygg)

	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 2 * time.Minute
	bo := backoff.WithContext(exp, ctx)
	err = backoff.RetryNotify(func() error {
		nodeID, twinID, err = registerNode(ctx, env, cl, sub, info)
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
			err := watch(ctx, env, cl, sub, info)
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
	sub substrate.Manager,
	info RegistrationInfo,
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
			if !yggNew.Equal(info.Ygg) {
				info = info.WithYggdrail(yggNew)
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
			_, _, err := registerNode(ctx, env, cl, sub, info)
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
	subMgr substrate.Manager,
	info RegistrationInfo,
) (nodeID, twinID uint32, err error) {
	var (
		mgr    = stubs.NewIdentityManagerStub(cl)
		netMgr = stubs.NewNetworkerStub(cl)
	)

	sub, err := subMgr.Substrate()
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to get substrate connection")
	}
	defer sub.Close()

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
		HRU: types.U64(info.Capacity.HRU),
		SRU: types.U64(info.Capacity.SRU),
		CRU: types.U64(info.Capacity.CRU),
		MRU: types.U64(info.Capacity.MRU),
	}

	location := substrate.Location{
		Longitude: fmt.Sprint(info.Location.Longitute),
		Latitude:  fmt.Sprint(info.Location.Latitude),
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

	twinID, err = ensureTwin(sub, sk, info.Ygg)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to ensure twin")
	}

	nodeID, err = sub.GetNodeByTwinID(twinID)

	create := substrate.Node{
		FarmID:      types.U32(env.FarmerID),
		TwinID:      types.U32(twinID),
		Resources:   resources,
		Location:    location,
		Country:     info.Location.Country,
		City:        info.Location.City,
		Interfaces:  interfaces,
		SecureBoot:  info.SecureBoot,
		Virtualized: info.Virtualized,
		BoardSerial: info.SerialNumber,
	}

	if errors.Is(err, substrate.ErrNotFound) {
		// create node
		nodeID, err = sub.CreateNode(id, create)

		return nodeID, twinID, err
	} else if err != nil {
		return 0, 0, errors.Wrapf(err, "failed to get node information for twin id: %d", twinID)
	}

	create.ID = types.U32(nodeID)

	// node exists. we validate everything is good
	// otherwise we update the node
	log.Debug().Uint32("node", nodeID).Msg("node already found on blockchain")
	current, err := sub.GetNode(nodeID)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "failed to get node with id: %d", nodeID)
	}

	if !create.Eq(current) {
		log.Debug().Msgf("node data have changing, issuing an update node: %+v", create)
		_, err := sub.UpdateNode(id, create)
		if err != nil {
			return 0, 0, errors.Wrapf(err, "failed to update node data with id: %d", nodeID)
		}
	}

	return uint32(nodeID), uint32(twinID), err
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

	subMgr, err := environment.GetSubstrate()
	if err != nil {
		return err
	}

	sk := ed25519.PrivateKey(mgr.PrivateKey(ctx))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return err
	}

	update := func(uptime uint64) (types.Hash, error) {
		sub, err := subMgr.Substrate()
		if err != nil {
			return types.Hash{}, err
		}
		defer sub.Close()
		return sub.UpdateNodeUptime(id, uptime)
	}

	for {
		uptime, err := host.Uptime()
		if err != nil {
			return errors.Wrap(err, "failed to get uptime")
		}
		log.Debug().Msg("updating node uptime")
		hash, err := update(uptime)
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
