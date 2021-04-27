package vm

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	chBin = "cloud-hypervisor"
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

	var fds []int
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
				fds = append(fds, idx)
				// fds[fd] = idx
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

	var argsList []string
	for k, vl := range args {
		argsList = append(argsList, k)
		argsList = append(argsList, vl...)
	}

	var fullArgs []string
	// setting setsid
	// without this the CH process will exit if vmd is stopped.
	// optimally, this should be done by the SysProcAttr
	// but we always get permission denied error and it's not
	// clear why. so for now we use busybox setsid command to do
	// this.
	fullArgs = append(fullArgs, "setsid", chBin)
	fullArgs = append(fullArgs, argsList...)
	cmd := exec.CommandContext(ctx, "busybox", fullArgs...)
	// TODO: always get permission denied when setting
	// sid with sys proc attr
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	// 	Setsid:     true,
	// 	Setpgid:    true,
	// 	Foreground: false,
	// 	Noctty:     true,
	// 	Setctty:    true,
	// }

	var toClose []io.Closer

	for _, tapindex := range fds {
		tap, err := os.OpenFile(filepath.Join("/dev", fmt.Sprintf("tap%d", tapindex)), os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		toClose = append(toClose, tap)
		cmd.ExtraFiles = append(cmd.ExtraFiles, tap)
	}

	defer func() {
		for _, c := range toClose {
			c.Close()
		}
	}()

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start cloud-hypervisor")
	}

	return cmd.Process.Release()
}
