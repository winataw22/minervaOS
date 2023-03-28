package dhcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type ProbeOutput struct {
	Subnet        string `json:"subnet"`
	Router        string `json:"router"`
	IP            string `json:"ip"`
	SourceAddress string `json:"siaddr"`
	DNS           string `json:"dns"`
	ServerID      string `json:"serverid"`
	BroadcastIP   string `json:"broadcast"`
	Lease         string `json:"lease"`
}

func (p *ProbeOutput) IPNet() (*net.IPNet, error) {
	mask := net.ParseIP(p.Subnet).To4()
	if mask == nil {
		return nil, fmt.Errorf("invalid subnet mask (%s)", p.Subnet)
	}
	ip := net.ParseIP(p.IP).To4()
	if ip == nil {
		return nil, fmt.Errorf("invalid ip  (%s)", p.IP)
	}

	return &net.IPNet{
		IP:   ip,
		Mask: net.IPMask(mask),
	}, nil
}

func Probe(ctx context.Context, inf string) (output ProbeOutput, err error) {
	// use udhcpc to prope the interface.
	// this depends on that the interface is UP
	cmd := exec.CommandContext(ctx, "udhcpc",
		"-i", inf, //the interface to prope
		"-q",       // exist once lease is optained
		"-f",       //foreground
		"-t", "20", //send 20 dhcp queries
		"-T", "1", // every second
		"-s", "/usr/share/udhcp/probe.script", // use the prope script
		"--now", // exit if lease is not obtained
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return output, errors.Wrapf(err, "failed to prope interface '%s': %s", inf, stderr.String())
	}

	log.Debug().Str("output", stdout.String()).Msg("output from dhcp proping")
	dec := json.NewDecoder(&stdout)
	if err := dec.Decode(&output); err != nil {
		buf, _ := io.ReadAll(dec.Buffered())
		str := stdout.String() + string(buf)
		return output, errors.Wrapf(err, "failed to decode proper output (%s)", str)
	}

	log.Debug().Msgf("output: %+v", output)
	return
}
