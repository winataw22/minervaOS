package yggdrasil

import (
	"context"
	"crypto/ed25519"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/zinit"
)

func EnsureYggdrasil(ctx context.Context, privateKey ed25519.PrivateKey, ns YggdrasilNamespace) (*YggServer, error) {
	// Filter out all the nodes from the same
	// segment so we do not just connect locally
	ips, err := ns.GetIPs() // returns ipv6 only
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ndmz public ipv6")
	}

	var ranges Ranges
	for _, ip := range ips {
		if ip.IP.IsGlobalUnicast() {
			ranges = append(ranges, ip)
		}
	}
	log.Info().Msgf("filtering out peers from ranges: %s", ranges)
	filter := Exclude(ranges)
	z := zinit.Default()

	cfg := GenerateConfig(privateKey)
	if err := cfg.FindPeers(ctx, filter); err != nil {
		return nil, err
	}

	server := NewYggServer(&cfg)
	if err := server.Ensure(z, ns.Name()); err != nil {
		return nil, err
	}

	gw, err := server.Gateway()
	if err != nil {
		return nil, errors.Wrap(err, "fail read yggdrasil subnet")
	}

	if err := ns.SetYggIP(gw, nil); err != nil {
		return nil, errors.Wrap(err, "fail to configure yggdrasil subnet gateway IP")
	}

	return server, nil
}
