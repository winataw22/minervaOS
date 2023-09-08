package iperf

import (
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	zinitService = "iperf"
	// IperfPort is the port for the iperf service
	IperfPort = 300
)

// Ensure creates an iperf zinit service and monitors it
func Ensure(z *zinit.Client) error {
	if _, err := z.Status(zinitService); err == nil {
		return nil
	}

	_, err := exec.LookPath("iperf")
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("ip netns exec public iperf -s -p %d", IperfPort)

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: cmd,
		After: []string{
			"networkd",
		},
	})

	if err != nil {
		return errors.Wrap(err, "failed to add iperf service")
	}

	return z.Monitor(zinitService)
}
