package registrar

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/geoip"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	tcUrl  = "http://zos.tf/terms/v0.1"
	tcHash = "9021d4dee05a661e2cb6838152c67f25" // not this is hash of the url not the document
)

type RegistrationInfo struct {
	Capacity     gridtypes.Capacity
	Location     geoip.Location
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

func (r *Registrar) registration(ctx context.Context, cl zbus.Client, env environment.Environment, info RegistrationInfo) (nodeID, twinID uint32, err error) {
	// we need to collect all node information here
	// - we already have capacity
	// - we get the location (will not change after initial registration)
	loc, err := geoip.Fetch()
	if err != nil {
		return 0, 0, errors.Wrap(err, "fetch location")
	}

	log.Debug().
		Uint64("cru", info.Capacity.CRU).
		Uint64("mru", uint64(info.Capacity.MRU)).
		Uint64("sru", uint64(info.Capacity.SRU)).
		Uint64("hru", uint64(info.Capacity.HRU)).
		Msg("node capacity")

	sub, err := environment.GetSubstrate()
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to create substrate client")
	}

	info = info.WithLocation(loc)

	nodeID, twinID, err = registerNode(ctx, env, cl, sub, info)

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
		Country:   info.Location.Country,
		City:      info.Location.City,
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

	twinID, err = ensureTwin(sub, sk)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to ensure twin")
	}

	nodeID, err = sub.GetNodeByTwinID(twinID)

	var serial substrate.OptionBoardSerial
	if len(info.SerialNumber) != 0 {
		serial = substrate.OptionBoardSerial{HasValue: true, AsValue: info.SerialNumber}
	}

	create := substrate.Node{
		FarmID:      types.U32(env.FarmID),
		TwinID:      types.U32(twinID),
		Resources:   resources,
		Location:    location,
		Interfaces:  interfaces,
		SecureBoot:  info.SecureBoot,
		Virtualized: info.Virtualized,
		BoardSerial: serial,
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

func ensureTwin(sub *substrate.Substrate, sk ed25519.PrivateKey) (uint32, error) {
	identity, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return 0, err
	}
	twinID, err := sub.GetTwinByPubKey(identity.PublicKey())
	if errors.Is(err, substrate.ErrNotFound) {
		return sub.CreateTwin(identity, "", nil)
	} else if err != nil {
		return 0, errors.Wrap(err, "failed to list twins")
	}

	return twinID, nil
}
