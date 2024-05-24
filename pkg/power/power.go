package power

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
)

type PowerServer struct {
	consumer         *events.RedisConsumer
	substrateGateway *stubs.SubstrateGatewayStub

	// enabled means the node can power off!
	enabled bool
	farm    pkg.FarmID
	node    uint32
	twin    uint32
	ut      *Uptime
}

func NewPowerServer(
	substrateGateway *stubs.SubstrateGatewayStub,
	consumer *events.RedisConsumer,
	enabled bool,
	farm pkg.FarmID,
	node uint32,
	twin uint32,
	ut *Uptime) (*PowerServer, error) {

	return &PowerServer{
		substrateGateway: substrateGateway,
		consumer:         consumer,
		enabled:          enabled,
		farm:             farm,
		node:             node,
		twin:             twin,
		ut:               ut,
	}, nil
}

const (
	DefaultWolBridge = "zos"
	PowerServerPort  = 8039
)

func EnsureWakeOnLan(ctx context.Context) (bool, error) {
	inf, err := bridge.Get(DefaultWolBridge)
	if err != nil {
		return false, errors.Wrap(err, "failed to get zos bridge")
	}

	nics, err := bridge.ListNics(inf, true)
	if err != nil {
		return false, errors.Wrap(err, "failed to list attached nics to zos bridge")
	}

	filtered := nics[:0]
	for _, nic := range nics {
		if nic.Type() == "device" {
			filtered = append(filtered, nic)
		}
	}

	if len(filtered) != 1 {
		return false, fmt.Errorf("zos bridge has multiple interfaces")
	}

	nic := filtered[0].Attrs().Name
	log.Info().Str("nic", nic).Msg("enabling wol on interface")
	support, err := ValueOfFlag(ctx, nic, SupportsWakeOn)

	if errors.Is(err, ErrFlagNotFound) {
		// no support for
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to detect support for wake on lan")
	}

	if !strings.Contains(support, string(MagicPacket)) {
		// no magic packet support either
		return false, nil
	}

	return true, SetWol(ctx, nic, MagicPacket)
}

func (p *PowerServer) syncSelf() error {
	power, err := p.substrateGateway.GetPowerTarget(context.Background(), p.node)
	if err != nil {
		return err
	}

	// power target is the state the node has to be in
	// while the node state is the actual state set by the node.

	// if target is up, and the node state is up, we do nothing
	// if target is up, but th state is down, we set the state to up and return
	// if target is down, we make sure state is down, then shutdown

	if power.Target.IsUp {
		if err := p.setNodePowerState(true); err != nil {
			return errors.Wrap(err, "failed to set state to up")
		}

		return nil
	}

	// now the target must be down.
	// we need to shutdown

	if err := p.setNodePowerState(false); err != nil {
		return errors.Wrap(err, "failed to set state to down")
	}

	// otherwise node need to get back to sleep.
	if err := p.shutdown(); err != nil {
		return errors.Wrap(err, "failed to issue shutdown")
	}

	return nil
}

func (p *PowerServer) powerUp(node *substrate.Node, reason string) error {
	log.Info().Uint32("node", uint32(node.ID)).Str("reason", reason).Msg("powering on node")

	mac := ""
	for _, inf := range node.Interfaces {
		if inf.Name == "zos" {
			mac = inf.Mac
			break
		}
	}
	if mac == "" {
		return fmt.Errorf("can't find mac address of node '%d'", node.ID)
	}

	for i := 0; i < 10; i++ {
		if err := exec.Command("ether-wake", "-i", "zos", mac).Run(); err != nil {
			log.Error().Err(err).Msg("failed to send WOL")
		}
		<-time.After(500 * time.Millisecond)
	}

	return nil
}

func (p *PowerServer) shutdown() error {
	if !p.enabled {
		log.Info().Msg("ignoring shutdown because power-management is not enabled")
		return nil
	}

	log.Info().Msg("shutting down node because of chain")
	if _, err := p.ut.SendNow(); err != nil {
		log.Error().Err(err).Msg("failed to send uptime before shutting down")
	}

	// is down!
	init := zinit.Default()
	err := init.Shutdown()

	if errors.Is(err, zinit.ErrNotSupported) {
		log.Info().Msg("node does not support shutdown. rebooting to update")
		return init.Reboot()
	}

	return err
}

func (p *PowerServer) event(event *pkg.PowerTargetChangeEvent) error {
	if event.FarmID != p.farm {
		return nil
	}

	log.Debug().
		Uint32("farm", uint32(p.farm)).
		Uint32("node", p.node).
		Msg("received power event for farm")

	node, err := p.substrateGateway.GetNode(context.Background(), event.NodeID)
	if err != nil {
		return err
	}

	if event.NodeID == p.node && event.Target.IsDown {
		// we need to shutdown!
		if err := p.setNodePowerState(false); err != nil {
			return errors.Wrap(err, "failed to set node power state to down")
		}

		return p.shutdown()
	} else if event.Target.IsDown {
		return nil
	}

	if event.Target.IsUp {
		log.Info().Uint32("target", event.NodeID).Msg("received an event to power up")
		return p.powerUp(&node, "target is up")
	}

	return nil
}

// setNodePowerState sets the node power state as provided or to up if power mgmt is
// not enabled on this node.
// this function makes sure to compare the state with on chain state to not do
// un-necessary transactions.
func (p *PowerServer) setNodePowerState(up bool) error {

	/*
		if power is not enabled, the node state should always be up
		otherwise update the state to the correct value

		| enabled | up | result|
		| 0       | 0  | 1     |
		| 0       | 1  | 1     |
		| 1       | 0  | 0     |
		| 1       | 1  | 1     |

		this simplifies as below:
	*/

	up = !p.enabled || up
	power, err := p.substrateGateway.GetPowerTarget(context.Background(), p.node)

	if err != nil {
		return errors.Wrap(err, "failed to check power state")
	}

	// only update the chain if it's different from actual value.
	if power.State.IsUp == up {
		return nil
	}

	log.Info().Bool("state", up).Msg("setting node power state")
	// this to make sure node state is fixed also for nodes
	_, err = p.substrateGateway.SetNodePowerState(context.Background(), up)
	return err
}

func (p *PowerServer) recv(ctx context.Context) error {
	log.Info().Msg("listening for power events")

	if err := p.syncSelf(); err != nil {
		return errors.Wrap(err, "failed to synchronize power status")
	}

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := p.consumer.PowerTargetChange(subCtx)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus events")
	}

	for event := range stream {
		if err := p.event(&event); err != nil {
			return errors.Wrap(err, "failed to process power event")
		}
	}

	// if we reach here it means stream was ended. this can only happen
	// if and only if the steam was over and that can only be via a ctx
	// cancel.
	return nil
}

// start processing time events.
func (p *PowerServer) events(ctx context.Context) error {
	// first thing we need to make sure we are not suppose to be powered
	// off, so we need to sync with grid
	// make sure at least one uptime was already sent
	_ = p.ut.Mark.Done(ctx)

	// if the stream loop fails for any reason retry
	// unless context was cancelled
	for {
		err := p.recv(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to process power events")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func (p *PowerServer) Start(ctx context.Context) error {
	return p.events(ctx)
}
