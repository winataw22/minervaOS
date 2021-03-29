package vm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Run run the machine with cloud-hypervisor
func (m *Machine) Run(ctx context.Context, socket, logs string) error {

	// build command line
	args := map[string][]string{
		"--kernel":    {m.Boot.Kernel},
		"--initramfs": {m.Boot.Initrd},
		"--cmdline":   {m.Boot.Args},

		"--cpus":   {m.Config.CPU.String()},
		"--memory": {m.Config.Mem.String()},

		"--log-file":   {logs},
		"--api-socket": {socket},
	}

	// disks
	if len(m.Disks) > 0 {
		var disks []string
		for _, disk := range m.Disks {
			disks = append(disks, disk.String())
		}
		args["--disk"] = disks
	}

	fds := make(map[int]int)
	if len(m.Interfaces) > 0 {
		var interfaces []string

		for _, nic := range m.Interfaces {
			typ, idx, err := nic.getType()
			if err != nil {
				return errors.Wrapf(err, "failed to detect interface type '%s'", nic.Tap)
			}
			if typ == InterfaceTAP {
				interfaces = append(interfaces, nic.asTap())
			} else if typ == InterfaceMACvTAP {
				// macvtap
				fd := len(fds) + 3
				fds[fd] = idx
				interfaces = append(interfaces, nic.asMACvTap(fd))
			} else {
				return fmt.Errorf("unsupported tap device type '%s'", nic.Tap)
			}
		}
		args["--net"] = interfaces
	}

	const debug = false
	if debug {
		args["--console"] = []string{"off"}
		args["--serial"] = []string{"tty"}
	}

	var tmp bytes.Buffer
	write := func() error {
		if _, err := tmp.WriteString("exec cloud-hypervisor"); err != nil {
			return err
		}

		for k, vs := range args {
			if _, err := tmp.WriteString(" "); err != nil {
				return err
			}
			if _, err := tmp.WriteString(k); err != nil {
				return err
			}
			for _, v := range vs {
				if _, err := tmp.WriteString(" "); err != nil {
					return err
				}
				if _, err := tmp.WriteString("'"); err != nil {
					return err
				}
				if _, err := tmp.WriteString(v); err != nil {
					return err
				}
				if _, err := tmp.WriteString("'"); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := write(); err != nil {
		return errors.Wrap(err, "exec script write error")
	}

	for fd, index := range fds {
		if _, err := tmp.WriteString(fmt.Sprintf(" %d<>/dev/tap%d", fd, index)); err != nil {
			return err
		}
	}

	log.Debug().Str("name", m.ID).Msg("starting machine")
	//the reason we do this shit is that we want the process to daemoinize the process in the back ground
	var cmd *exec.Cmd
	if debug {
		cmd = exec.CommandContext(ctx, "ash", "-c", tmp.String())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
	} else {
		cmd = exec.CommandContext(ctx, "ash", "-c", fmt.Sprintf("%s >%s 2>&1 &", tmp.String(), logs+".out"))
	}

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to start cloud-hypervisor")
	}

	return nil
}
