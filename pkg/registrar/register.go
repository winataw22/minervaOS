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
	"github.com/threefoldtech/zos/pkg"
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
	// List of gpus short name
	GPUs map[string]interface{}
}

func (r RegistrationInfo) WithGPU(short string) RegistrationInfo {
	if r.GPUs == nil {
		r.GPUs = make(map[string]interface{})
	}

	r.GPUs[short] = nil
	return r
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

	info = info.WithLocation(loc)

	nodeID, twinID, err = registerNode(ctx, env, cl, info)
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
	info RegistrationInfo,
) (nodeID, twinID uint32, err error) {
	var (
		mgr              = stubs.NewIdentityManagerStub(cl)
		netMgr           = stubs.NewNetworkerStub(cl)
		substrateGateway = stubs.NewSubstrateGatewayStub(cl)
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
		HRU: types.U64(info.Capacity.HRU),
		SRU: types.U64(info.Capacity.SRU),
		CRU: types.U64(info.Capacity.CRU),
		MRU: types.U64(info.Capacity.MRU),
	}

	location := substrate.Location{
		Longitude: fmt.Sprint(info.Location.Longitude),
		Latitude:  fmt.Sprint(info.Location.Latitude),
		Country:   info.Location.Country,
		City:      info.Location.City,
	}

	log.Info().Str("id", mgr.NodeID(ctx).Identity()).Msg("start registration of the node")
	log.Info().Msg("registering node on blockchain")

	sk := ed25519.PrivateKey(mgr.PrivateKey(ctx))

	if _, err := substrateGateway.EnsureAccount(ctx, env.ActivationURL, tcUrl, tcHash); err != nil {
		return 0, 0, errors.Wrap(err, "failed to ensure account")
	}

	twinID, err = ensureTwin(ctx, substrateGateway, sk)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to ensure twin")
	}
	var subErr pkg.SubstrateError
	nodeID, subErr = substrateGateway.GetNodeByTwinID(ctx, twinID)

	var serial substrate.OptionBoardSerial
	if len(info.SerialNumber) != 0 {
		serial = substrate.OptionBoardSerial{HasValue: true, AsValue: info.SerialNumber}
	}

	real := substrate.Node{
		FarmID:      types.U32(env.FarmID),
		TwinID:      types.U32(twinID),
		Resources:   resources,
		Location:    location,
		Interfaces:  interfaces,
		SecureBoot:  info.SecureBoot,
		Virtualized: info.Virtualized,
		BoardSerial: serial,
	}

	var onChain substrate.Node
	if subErr.IsCode(pkg.CodeNotFound) {
		// node not found, create node
		nodeID, err = substrateGateway.CreateNode(ctx, real)
		if err != nil {
			return 0, 0, errors.Wrap(err, "failed to create node on chain")
		}

	} else if subErr.IsError() {
		// other error occurred
		return 0, 0, errors.Wrapf(subErr.Err, "failed to get node information for twin id: %d", twinID)
	} else {
		// node exists
		onChain, err = substrateGateway.GetNode(ctx, nodeID)
		if err != nil {
			return 0, 0, errors.Wrapf(err, "failed to get node with id: %d", nodeID)
		}

		// ignore virt-what value if the node is marked as real on the chain
		if !onChain.Virtualized {
			real.Virtualized = false
		}
	}

	real.ID = types.U32(nodeID)

	// node exists. we validate everything is good
	// otherwise we update the node
	log.Debug().Uint32("node", nodeID).Msg("node already found on blockchain")

	if !real.Eq(&onChain) {
		log.Debug().Msgf("node data have changing, issuing an update node: %+v", real)
		_, err := substrateGateway.UpdateNode(ctx, real)
		if err != nil {
			return 0, 0, errors.Wrapf(err, "failed to update node data with id: %d", nodeID)
		}
	}

	return nodeID, twinID, err
}

func ensureTwin(ctx context.Context, substrateGateway *stubs.SubstrateGatewayStub, sk ed25519.PrivateKey) (uint32, error) {
	identity, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return 0, err
	}
	twinID, subErr := substrateGateway.GetTwinByPubKey(ctx, identity.PublicKey())
	if subErr.IsCode(pkg.CodeNotFound) {
		return substrateGateway.CreateTwin(ctx, "", nil)
	} else if subErr.IsError() {
		return 0, errors.Wrap(subErr.Err, "failed to list twins")
	}

	return twinID, nil
}
