package networkd

import (
	"context"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func ensureHostFw(ctx context.Context) error {
	log.Info().Msg("ensuring existing host nft rules")

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c",
		`nft 'add table inet filter'
nft 'add chain inet filter input   { type filter hook input priority filter; policy accept; }'
nft 'add chain inet filter forward { type filter hook input priority filter; policy accept; }'
nft 'add chain inet filter output  { type filter hook input priority filter; policy accept; }'

nft 'add table arp filter'
nft 'add chain arp filter input  { type filter hook input priority filter; policy accept; }'
nft 'add chain arp filter output { type filter hook input priority filter; policy accept; }'

nft 'add table bridge filter'
nft 'add chain bridge filter input   { type filter hook input priority filter; policy accept; }'
nft 'add chain bridge filter forward { type filter hook input priority filter; policy accept; }'
nft 'add chain bridge filter prerouting { type filter hook prerouting priority filter; policy accept; }'
nft 'add chain bridge filter postrouting { type filter hook postrouting priority filter; policy accept; }'
nft 'add chain bridge filter output  { type filter hook input priority filter; policy accept; }'
nft 'flush chain bridge filter forward'
# nft 'add rule bridge filter forward icmpv6 type nd-router-advert drop'
# nft 'add rule bridge filter forward ip6 version 6 udp sport 547 drop'
# nft 'add rule bridge filter forward ip version 4 udp sport 67 drop'
`)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "could not set up host nft rules")
	}

	return nil
}
