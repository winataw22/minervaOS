package networkd

import (
	"context"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func ensureHostFw(ctx context.Context) error {
	log.Info().Msg("ensuring existing host nft rules")
	cmd := exec.CommandContext(ctx, "nft", "list", "ruleset")
	out, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "could not load existing nft rules")
	}
	outs := strings.TrimSpace(string(out))

	// there are already rules in place, nothing to do here
	if outs != "" {
		log.Info().Msg("found existing nft rules in host")
		return nil
	}

	cmd = exec.CommandContext(ctx, "/bin/sh", "-c",
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
nft 'add chain bridge filter output  { type filter hook input priority filter; policy accept; }'`)

	if err = cmd.Run(); err != nil {
		return errors.Wrap(err, "could not set up host nft rules")
	}

	return nil
}
