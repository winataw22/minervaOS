package dhcp

import (
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// Probe is used to do some DHCP request on a interface
type Probe struct {
	cmd *exec.Cmd
}

// NewProbe returns a Probe
func NewProbe() *Probe {
	return &Probe{}
}

// Start starts the DHCP client process
func (d *Probe) Start(inf string) error {

	d.cmd = exec.Command("udhcpc",
		"-f", //foreground
		"-i", inf,
		"-t", "20", //send 20 dhcp queries
		"-T", "1", // every second
		"-s", "/usr/share/udhcp/simple.script",
		"-p", fmt.Sprintf("/run/udhcpc.%s.pid", inf),
		"--now", // exit if lease is not obtained
	)

	log.Debug().Msgf("start udhcp: %v", d.cmd.String())
	if err := d.cmd.Start(); err != nil {
		return err
	}

	return nil
}

// Stop kills the DHCP client process
func (d *Probe) Stop() error {
	if d.cmd.ProcessState != nil && d.cmd.ProcessState.Exited() {
		return nil
	}

	if err := d.cmd.Process.Kill(); err != nil {
		log.Error().Err(err).Msg("could not kill proper")
		return err
	}

	_ = d.cmd.Wait()

	return nil
}
