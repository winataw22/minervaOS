package primitives

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/jbenet/go-base58"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// networkProvision is entry point to provision a network
func (p *Provisioner) networkProvisionImpl(ctx context.Context, reservation *provision.Reservation) error {
	nr := pkg.NetResource{}
	if err := json.Unmarshal(reservation.Data, &nr); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	if err := validateNR(nr); err != nil {
		return fmt.Errorf("validation of the network resource failed: %w", err)
	}

	nr.NetID = networkID(reservation.User, nr.Name)

	mgr := stubs.NewNetworkerStub(p.zbus)
	log.Debug().Str("network", fmt.Sprintf("%+v", nr)).Msg("provision network")

	_, err := mgr.CreateNR(nr)
	if err != nil {
		return errors.Wrapf(err, "failed to create network resource for network %s", nr.NetID)
	}

	return nil
}

func (p *Provisioner) networkProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return nil, p.networkProvisionImpl(ctx, reservation)
}

func (p *Provisioner) networkDecommission(ctx context.Context, reservation *provision.Reservation) error {
	mgr := stubs.NewNetworkerStub(p.zbus)

	network := &pkg.NetResource{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	network.NetID = networkID(reservation.User, network.Name)

	if err := mgr.DeleteNR(*network); err != nil {
		return fmt.Errorf("failed to delete network resource: %w", err)
	}
	return nil
}

func networkID(userID, name string) pkg.NetID {
	buf := bytes.Buffer{}
	buf.WriteString(userID)
	buf.WriteString(name)
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return pkg.NetID(string(b))
}

func validateNR(nr pkg.NetResource) error {

	if nr.NetID == "" {
		return fmt.Errorf("network ID cannot be empty")
	}

	if nr.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	if nr.NetworkIPRange.Nil() {
		return fmt.Errorf("network IP range cannot be empty")
	}

	if nr.NodeID == "" {
		return fmt.Errorf("network resource node ID cannot empty")
	}
	if nr.Subnet.IP == nil {
		return fmt.Errorf("network resource subnet cannot empty")
	}

	if nr.WGPrivateKey == "" {
		return fmt.Errorf("network resource wireguard private key cannot empty")
	}

	if nr.WGPublicKey == "" {
		return fmt.Errorf("network resource wireguard public key cannot empty")
	}

	if nr.WGListenPort == 0 {
		return fmt.Errorf("network resource wireguard listen port cannot empty")
	}

	for _, peer := range nr.Peers {
		if err := validatePeer(peer); err != nil {
			return err
		}
	}

	return nil
}

func validatePeer(p pkg.Peer) error {
	if p.WGPublicKey == "" {
		return fmt.Errorf("peer wireguard public key cannot empty")
	}

	if p.Subnet.Nil() {
		return fmt.Errorf("peer wireguard subnet cannot empty")
	}

	if len(p.AllowedIPs) <= 0 {
		return fmt.Errorf("peer wireguard allowedIPs cannot empty")
	}
	return nil
}
