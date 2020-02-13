package vm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v3"
	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
)

const (
	// FCSockDir where vm firecracker sockets are kept
	FCSockDir = "/var/run/firecracker"

	defaultKernelArgs = "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules"
)

// vmModuleImpl implements the VMModule interface
type vmModuleImpl struct {
	root string
}

var (
	_ pkg.VMModule = (*vmModuleImpl)(nil)

	errScanFound = fmt.Errorf("found")
)

// NewVMModule creates a new instance of vm manager
func NewVMModule(root string) (pkg.VMModule, error) {
	if err := os.MkdirAll(FCSockDir, 0755); err != nil {
		return nil, err
	}

	return &vmModuleImpl{
		root: root,
	}, nil
}

func (m *vmModuleImpl) makeDevices(vm *pkg.VM) ([]Drive, error) {
	var drives []Drive
	for i, disk := range vm.Disks {
		id := fmt.Sprintf("%d", i+2)

		drives = append(drives, Drive{
			ID:         id,
			ReadOnly:   disk.ReadOnly,
			RootDevice: disk.Root,
			Path:       disk.Path,
		})
	}

	return drives, nil
}

func (m *vmModuleImpl) machineRoot(id string) string {
	return filepath.Join(m.root, "firecracker", id)
}

func (m *vmModuleImpl) socket(id string) string {
	return filepath.Join(m.machineRoot(id), "root", "api.socket")
}

func (m *vmModuleImpl) Exists(id string) bool {
	_, err := m.find(id)
	return err == nil
}

func (m *vmModuleImpl) cleanFs(id string) error {
	root := filepath.Join(m.machineRoot(id), "root")

	files, err := ioutil.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	for _, entry := range files {
		if entry.IsDir() {
			continue
		}

		// we try to unmount every file in the directory
		// because it's faster than trying to find exactly
		// what files are mounted under this location.
		path := filepath.Join(root, entry.Name())
		err := syscall.Unmount(
			path,
			syscall.MNT_DETACH,
		)

		if err != nil {
			log.Warn().Err(err).Str("file", path).Msg("failed to unmount")
		}
	}

	return os.RemoveAll(m.machineRoot(id))
}

func (m *vmModuleImpl) makeNetwork(vm *pkg.VM) (iface Interface, cmdline string, err error) {
	netIP := vm.Network.AddressCIDR

	nic := Interface{
		ID:  "eth0",
		Tap: vm.Network.Tap,
		Mac: vm.Network.MAC,
	}

	dns0 := ""
	dns1 := ""
	if len(vm.Network.Nameservers) > 0 {
		dns0 = vm.Network.Nameservers[0].String()
	}
	if len(vm.Network.Nameservers) > 1 {
		dns1 = vm.Network.Nameservers[1].String()
	}

	cmdline = fmt.Sprintf("ip=%s::%s:%s:::off:%s:%s:",
		netIP.IP.String(),
		vm.Network.GatewayIP.String(),
		net.IP(netIP.Mask).String(),
		dns0,
		dns1,
	)

	return nic, cmdline, nil
}

func (m *vmModuleImpl) tail(path string) (string, error) {
	// fetch 2k of bytes from the path ?
	// TODO: implement a better tail algo.

	const (
		tail = 2 * 1024 // 2K
	)

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return "no logs available", nil
	} else if err != nil {
		return "", errors.Wrapf(err, "failed to tail file: %s", path)
	}

	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", errors.Wrapf(err, "fail to stat %s", f.Name())
	}
	offset := info.Size()
	if offset > tail {
		offset = tail
	}

	_, err = f.Seek(-offset, 2)
	if err != nil {
		return "", errors.Wrapf(err, "failed to seek file: %s", path)
	}

	logs, err := ioutil.ReadAll(f)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read logs from: %s", path)
	}

	return string(logs), nil
}

func (m *vmModuleImpl) withLogs(path string, err error) error {
	if err == nil {
		return nil
	}

	logs, tailErr := m.tail(path)
	if tailErr != nil {
		return errors.Wrapf(err, "failed to tail machine logs: %s", tailErr)
	}

	return errors.Wrapf(err, string(logs))
}

// Run vm
func (m *vmModuleImpl) Run(vm pkg.VM) error {
	if err := vm.Validate(); err != nil {
		return errors.Wrap(err, "machine configuration validation failed")
	}

	ctx := context.Background()

	if m.Exists(vm.Name) {
		return fmt.Errorf("a vm with same name already exists")
	}

	// make sure to clean up previous roots just in case
	if err := m.cleanFs(vm.Name); err != nil {
		return err
	}

	devices, err := m.makeDevices(&vm)
	if err != nil {
		return err
	}

	var kargs strings.Builder
	kargs.WriteString(vm.KernelArgs)
	if kargs.Len() == 0 {
		kargs.WriteString(defaultKernelArgs)
	}

	nic, args, err := m.makeNetwork(&vm)
	if err != nil {
		return err
	}

	if kargs.Len() != 0 {
		kargs.WriteRune(' ')
	}

	kargs.WriteString(args)

	machine := Machine{
		ID: vm.Name,
		Boot: Boot{
			Kernel: vm.KernelImage,
			Initrd: vm.InitrdImage,
			Args:   kargs.String(),
		},
		Config: Config{
			CPU:       vm.CPU,
			Mem:       vm.Memory,
			HTEnabled: false,
		},
		Interfaces: []Interface{
			nic,
		},
		Drives: devices,
	}

	defer func() {
		if err != nil {
			m.Delete(machine.ID)
		}
	}()

	logFile := machine.Log(m.root)

	if err = machine.Start(ctx, m.root); err != nil {
		return m.withLogs(logFile, err)
	}

	check := func() error {
		if !m.Exists(machine.ID) {
			return fmt.Errorf("failed to spawn vm machine process '%s'", machine.ID)
		}
		//TODO: check unix connection
		socket := m.socket(machine.ID)
		con, err := net.Dial("unix", socket)
		if err != nil {
			return err
		}

		con.Close()
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	// wait for the machine to answer
	if err = backoff.Retry(check, backoff.WithContext(backoff.NewConstantBackOff(2*time.Second), ctx)); err != nil {
		return m.withLogs(logFile, err)
	}

	return nil
}

func (m *vmModuleImpl) Inspect(name string) (pkg.VMInfo, error) {
	if !m.Exists(name) {
		return pkg.VMInfo{}, fmt.Errorf("machine '%s' does not exist", name)
	}

	client := firecracker.NewClient(m.socket(name), nil, false)
	cfg, err := client.GetMachineConfiguration()
	if err != nil {
		return pkg.VMInfo{}, errors.Wrap(err, "failed to get machine configuration")
	}

	return pkg.VMInfo{
		CPU:       *cfg.Payload.VcpuCount,
		Memory:    *cfg.Payload.MemSizeMib,
		HtEnabled: *cfg.Payload.HtEnabled,
	}, nil
}

func (m *vmModuleImpl) find(name string) (int, error) {
	const (
		proc   = "/proc"
		search = "/firecracker"
	)
	idArg := name
	result := 0
	err := filepath.Walk(proc, func(path string, info os.FileInfo, _ error) error {
		if path == proc {
			// assend into /proc
			return nil
		}

		dir, name := filepath.Split(path)

		if filepath.Clean(dir) != proc {
			// this to make sure we only scan first level
			return filepath.SkipDir
		}

		pid, err := strconv.Atoi(name)
		if err != nil {
			//not a number
			return nil //continue scan
		}
		cmd, err := ioutil.ReadFile(filepath.Join(path, "cmdline"))
		if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return err
		}

		parts := bytes.Split(cmd, []byte{0})
		if string(parts[0]) != search {
			return nil
		}

		// a firecracker instance, now find id
		for _, part := range parts {
			if string(part) == idArg {
				// a hit
				result = pid
				// this is to stop the scan.
				return errScanFound
			}
		}

		return nil
	})

	if err == errScanFound {
		return result, nil
	} else if err != nil {
		return 0, err
	}

	return 0, fmt.Errorf("vm '%s' not found", name)
}

func (m *vmModuleImpl) Delete(name string) error {
	defer m.cleanFs(name)

	pid, err := m.find(name)
	if err != nil {
		// machine already gone
		return nil
	}

	client := firecracker.NewClient(m.socket(name), nil, false)
	// timeout is request timeout, not machine timeout to shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()
	action := models.InstanceActionInfoActionTypeSendCtrlAltDel
	info := models.InstanceActionInfo{
		ActionType: &action,
	}

	now := time.Now()

	const (
		termAfter = 5 * time.Second
		killAfter = 10 * time.Second
	)

	_, err = client.CreateSyncAction(ctx, &info)
	if err != nil {
		return err
	}

	for {
		if !m.Exists(name) {
			return nil
		}

		if time.Since(now) > termAfter {
			syscall.Kill(pid, syscall.SIGTERM)
		}

		if time.Since(now) > killAfter {
			syscall.Kill(pid, syscall.SIGKILL)
			break
		}

		<-time.After(1 * time.Second)
	}

	return nil
}
