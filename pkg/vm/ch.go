package vm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/google/shlex"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
)

const (
	chBin           = "cloud-hypervisor"
	cloudConsoleBin = "cloud-console"
)

// startCloudConsole Starts the cloud console for the vm on it's private network ip
func (m *Machine) startCloudConsole(ctx context.Context, namespace string, networkAddr net.IPNet, machineIP net.IPNet, ptyPath string, logs string) (string, error) {
	ipv4 := machineIP.IP.To4()
	if ipv4 == nil {
		return "", fmt.Errorf("invalid vm ip address (%s) not ipv4", machineIP.IP.String())
	}
	port := 20000 + uint16(ipv4[3])
	if port == math.MaxUint16 {
		// this should be impossible since a byte max value is 512 hence 20_000 + 512 can never be over
		// max of uint16
		return "", fmt.Errorf("couldn't start cloud console port number exceeds %d", port)
	}
	args := []string{
		"setsid",
		"ip",
		"netns",
		"exec", namespace,
		cloudConsoleBin,
		ptyPath,
		networkAddr.IP.String(),
		fmt.Sprint(port),
		logs,
	}

	log.Debug().Msgf("running cloud-console : %+v", args)

	cmd := exec.CommandContext(ctx, "busybox", args...)
	if err := cmd.Start(); err != nil {
		return "", errors.Wrap(err, "failed to start cloud-hypervisor")
	}
	if err := m.release(cmd.Process); err != nil {
		return "", err
	}
	consoleURL := fmt.Sprintf("%s:%d", networkAddr.IP.String(), port)
	return consoleURL, nil
}

// Run run the machine with cloud-hypervisor
func (m *Machine) Run(ctx context.Context, socket, logs string) (pkg.MachineInfo, error) {
	_ = os.Remove(socket)

	// build command line
	args := map[string][]string{
		"--kernel":  {m.Boot.Kernel},
		"--cmdline": {m.Boot.Args},

		"--cpus":   {m.Config.CPU.String()},
		"--memory": {fmt.Sprintf("%s,shared=on", m.Config.Mem.String())},

		"--console":    {"off"},
		"--serial":     {"pty"}, // we use pty here for the cloud console to be able to read the vm console, in case of debuging or we need stdout logging we use tty
		"--api-socket": {socket},
	}
	var err error
	var pids []int
	defer func() {
		if err != nil {
			for _, pid := range pids {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	}()

	var filesystems []string
	for i, fs := range m.FS {
		socket := filepath.Join("/var", "run", fmt.Sprintf("virtio-%s-%d.socket", m.ID, i))
		var pid int
		pid, err = m.startFs(socket, fs.Path)
		if err != nil {
			return pkg.MachineInfo{}, err
		}
		pids = append(pids, pid)
		filesystems = append(filesystems, fmt.Sprintf("tag=%s,socket=%s", fs.Tag, socket))
	}

	if len(filesystems) > 0 {
		args["--fs"] = filesystems
	}
	if err := m.buildZosRC(); err != nil {
		return pkg.MachineInfo{}, errors.Wrap(err, "couldn't build zosrc")
	}
	if m.Boot.Initrd != "" {
		args["--initramfs"] = []string{m.Boot.Initrd}
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
			var typ InterfaceType
			typ, _, err = nic.getType()
			if err != nil {
				return pkg.MachineInfo{}, errors.Wrapf(err, "failed to detect interface type '%s'", nic.Tap)
			}
			if typ == InterfaceTAP {
				interfaces = append(interfaces, nic.asTap())
			} else {
				err = fmt.Errorf("unsupported tap device type '%s'", nic.Tap)
				return pkg.MachineInfo{}, err
			}
		}
		args["--net"] = interfaces
	}

	const debug = false
	if debug {
		args["--serial"] = []string{"tty"}
	}

	var argsList []string
	for k, vl := range args {
		argsList = append(argsList, k)
		argsList = append(argsList, vl...)
	}

	var fullArgs []string

	// open the log file for full stdout/stderr piping. The file is
	// open in append mode so we can safely truncate the file on the disk
	// to save up storage.
	logFd, err := os.OpenFile(logs, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return pkg.MachineInfo{}, err
	}
	defer logFd.Close()

	// setting setsid
	// without this the CH process will exit if vmd is stopped.
	// optimally, this should be done by the SysProcAttr
	// but we always get permission denied error and it's not
	// clear why. so for now we use busybox setsid command to do
	// this.
	fullArgs = append(fullArgs, "setsid", chBin)
	fullArgs = append(fullArgs, argsList...)
	log.Debug().Msgf("ch: %+v", fullArgs)

	cmd := exec.CommandContext(ctx, "busybox", fullArgs...)
	cmd.Stdout = logFd
	cmd.Stderr = logFd

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
		var tap *os.File
		tap, err = os.OpenFile(filepath.Join("/dev", fmt.Sprintf("tap%d", tapindex)), os.O_RDWR, 0600)
		if err != nil {
			return pkg.MachineInfo{}, err
		}
		toClose = append(toClose, tap)
		cmd.ExtraFiles = append(cmd.ExtraFiles, tap)
	}

	defer func() {
		for _, c := range toClose {
			c.Close()
		}
	}()

	if err = cmd.Start(); err != nil {
		return pkg.MachineInfo{}, errors.Wrap(err, "failed to start cloud-hypervisor")
	}

	if err = m.release(cmd.Process); err != nil {
		return pkg.MachineInfo{}, err
	}

	if err := m.waitAndAdjOom(ctx, m.ID, socket); err != nil {
		return pkg.MachineInfo{}, err
	}
	client := NewClient(socket)
	vmData, err := client.Inspect(ctx)

	if err != nil {
		return pkg.MachineInfo{}, errors.Wrapf(err, "failed to Inspect vm with id: '%s'", m.ID)
	}
	consoleURL := ""
	for _, ifc := range m.Interfaces {
		if ifc.Console != nil {
			consoleURL, err = m.startCloudConsole(ctx, ifc.Console.Namespace, ifc.Console.NetworkAddr, ifc.Console.IP, vmData.PTYPath, logs)
			if err != nil {
				log.Error().Err(err).Str("vm", m.ID).Msg("failed to start cloud-console for vm")
			}
		}
	}

	return pkg.MachineInfo{ConsoleURL: consoleURL}, nil
}

func (m *Machine) waitAndAdjOom(ctx context.Context, name string, socket string) error {
	check := func() error {
		if _, err := Find(name); err != nil {
			return fmt.Errorf("failed to spawn vm machine process '%s'", name)
		}

		con, err := net.Dial("unix", socket)
		if err != nil {
			return err
		}

		con.Close()
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := backoff.RetryNotify(
		check,
		backoff.WithContext(
			backoff.NewConstantBackOff(2*time.Second),
			ctx,
		),
		func(err error, d time.Duration) {
			log.Debug().Err(err).Str("id", name).Msg("vm is not up yet")
		}); err != nil {

		return err
	}

	ps, err := Find(name)
	if err != nil {
		return errors.Wrapf(err, "failed to find vm with id '%s'", name)
	}

	if err := os.WriteFile(filepath.Join("/proc/", fmt.Sprint(ps.Pid), "oom_adj"), []byte("-17"), 0644); err != nil {
		return errors.Wrapf(err, "failed to update oom priority for machine '%s' (PID: %d)", name, ps.Pid)
	}

	return nil
}

func (m *Machine) findVirtioFsMount() (*VirtioFS, error) {
	var root *VirtioFS
	for i := range m.FS {
		fs := &m.FS[i]
		if fs.Tag == virtioRootFsTag {
			root = fs
			break
		}
	}
	if root == nil {
		return nil, errors.New("no virtiofs mounts found")
	}
	return root, nil
}

func (m *Machine) buildZosRC() error {
	if len(m.Environment) == 0 && m.Entrypoint == "" {
		// nothing to add in .zosrc
		return nil
	}

	fs, err := m.findVirtioFsMount()
	if err != nil {
		return err
	}
	root := fs.Path
	stat, err := os.Stat(root)
	if err != nil {
		return errors.Wrap(err, "failed to stat vm rootfs")
	}
	if !stat.IsDir() {
		return fmt.Errorf("vm rootfs is not a directory")
	}

	file, err := os.OpenFile(
		filepath.Join(root, ".zosrc"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return errors.Wrap(err, "failed to open zosrc file")
	}

	defer file.Close()
	if err := m.appendEnv(file); err != nil {
		return errors.Wrap(err, "couldn't append environment variables to zosrc")
	}
	if err := m.appendEntrypoint(file); err != nil {
		return errors.Wrap(err, "couldn't append entrypoint data to zosrc")
	}
	return nil
}

func (m *Machine) appendEnv(file io.Writer) error {
	for k, v := range m.Environment {
		if _, err := fmt.Fprintf(file, "export %s=%s\n", k, quote(v)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Machine) appendEntrypoint(file io.Writer) error {
	parts, err := shlex.Split(m.Entrypoint)
	if err != nil {
		return errors.Wrap(err, "invalid entrypoint")
	}
	if len(parts) != 0 {
		if _, err := fmt.Fprintf(file, "init=%s\n", quote(parts[0])); err != nil {
			return err
		}
	}
	if len(parts) > 1 {
		var buf bytes.Buffer
		buf.WriteString("set --")
		for _, part := range parts[1:] {
			buf.WriteRune(' ')
			buf.WriteString(quote(part))
		}
		if _, err := fmt.Fprint(file, buf.String()); err != nil {
			return err
		}
	}
	return nil
}

func (m *Machine) startFs(socket, path string) (int, error) {
	cmd := exec.Command("busybox", "setsid",
		"virtiofsd-rs",
		"--socket", socket,
		"--shared-dir", path,
	)

	if err := cmd.Start(); err != nil {
		return 0, errors.Wrap(err, "failed to start virtiofsd-")
	}

	return cmd.Process.Pid, m.release(cmd.Process)
}

func (m *Machine) release(ps *os.Process) error {
	pid := ps.Pid
	go func() {
		ps, err := os.FindProcess(pid)
		if err != nil {
			log.Error().Err(err).Msgf("failed to find process with id: %d", pid)
			return
		}

		_, _ = ps.Wait()
	}()

	if err := ps.Release(); err != nil {
		return errors.Wrap(err, "failed to release cloud-hypervisor process")
	}

	return nil
}

// transpiled from https://github.com/python/cpython/blob/3.10/Lib/shlex.py#L325
func quote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
