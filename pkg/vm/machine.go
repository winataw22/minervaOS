package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

// Boot config struct
type Boot struct {
	Kernel string `json:"kernel_image_path"`
	Initrd string `json:"initrd_path,omitempty"`
	Args   string `json:"boot_args"`
}

// Drive struct
type Drive struct {
	ID         string `json:"drive_id"`
	Path       string `json:"path_on_host"`
	RootDevice bool   `json:"is_root_device"`
	ReadOnly   bool   `json:"is_read_only"`
}

// Interface nic struct
type Interface struct {
	ID  string `json:"iface_id"`
	Tap string `json:"host_dev_name"`
	Mac string `json:"guest_mac,omitempty"`
}

// Config struct
type Config struct {
	CPU       uint8 `json:"vcpu_count"`
	Mem       int64 `json:"mem_size_mib"`
	HTEnabled bool  `json:"ht_enabled"`
}

// Machine struct
type Machine struct {
	ID         string      `json:"-"`
	Boot       Boot        `json:"boot-source"`
	Drives     []Drive     `json:"drives"`
	Interfaces []Interface `json:"network-interfaces"`
	Config     Config      `json:"machine-config"`
}

func (m *Machine) root(base string) string {
	return filepath.Join(base, "firecracker", m.ID, "root")
}

func mount(src, dest string) error {
	if filepath.Clean(src) == filepath.Clean(dest) {
		// nothing to do here
		return nil
	}

	f, err := os.Create(dest)
	if err != nil {
		return errors.Wrapf(err, "failed to touch file: %s", dest)
	}

	f.Close()

	if err := syscall.Mount(src, dest, "", syscall.MS_BIND, ""); err != nil {
		return errors.Wrapf(err, "failed to mount '%s' > '%s'", src, dest)
	}

	return nil
}

// Jail will move files to machine root and update a "jailed"
// copy of the config to reference correct files.
func (m *Machine) jail(root string) (Machine, error) {
	cfg := *m
	if err := os.MkdirAll(root, 0755); err != nil {
		return cfg, err
	}

	rooted := func(f string) string {
		return filepath.Join(root, filepath.Base(f))
	}

	for _, str := range []*string{&cfg.Boot.Kernel, &cfg.Boot.Initrd} {
		file := *str
		if len(file) == 0 {
			continue
		}

		// mount kernel
		if err := mount(file, rooted(file)); err != nil {
			return cfg, err
		}

		*str = filepath.Base(file)
	}

	// mount drives
	for i, drive := range cfg.Drives {
		name := filepath.Base(drive.Path)
		if err := mount(drive.Path, rooted(drive.Path)); err != nil {
			return cfg, err
		}

		m.Drives[i].Path = name
	}

	return cfg, nil
}

// Start starts the machine.
func (m *Machine) Start(ctx context.Context, base string) error {
	root := m.root(base)
	if err := os.MkdirAll(root, 0755); err != nil {
		return errors.Wrap(err, "failed to create machine root")
	}

	jailed, err := m.jail(root)
	if err != nil {
		return errors.Wrap(err, "failed to jail files")
	}

	cfg, err := os.Create(filepath.Join(root, "config.json"))
	if err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	defer cfg.Close()
	enc := json.NewEncoder(cfg)
	if err := enc.Encode(jailed); err != nil {
		return err
	}

	cfg.Close()

	return jailed.exec(ctx, base)
}

// Log returns machine log file path
func (m *Machine) Log(base string) string {
	return filepath.Join(m.root(base), "machine.log")
}

func (m *Machine) exec(ctx context.Context, base string) error {
	// prepare command
	// because the --daemonize flag does not work as expected
	// we are daemonizing with `ash and &` so we can use cmd.Run().
	// the reason we don't want to use 'cmd.Start' instead of 'cmd.Run'
	// is that if we do 'Start' and then the process exited for some reason
	// we will endup with 'zombi' process because vmd did not 'wait' for the
	// process exit. hence daemonizing is required.
	// due to issues with jailer --daemonize flag, we use the ash trick below

	jailerBin, err := exec.LookPath("jailer")
	if err != nil {
		return err
	}

	fcBin, err := exec.LookPath("firecracker")
	if err != nil {
		return err
	}

	args := []string{
		jailerBin,
		"--id", m.ID,
		"--uid", "0", "--gid", "0",
		"--chroot-base-dir", base, // this stupid flag creates so many layers but is needed
		"--exec-file", fcBin,
		"--node", "0",
		"--", // fc flags starts here
		"--config-file", "/config.json",
		"--api-sock", "/api.socket",
	}

	const (
		// if this is enabled machine will start in the
		// foreground. It's then required that vmd
		// was also started from the cmdline not as a
		// daemon so you have access to machine console

		testing = false
	)

	logFile := m.Log(base)

	var cmd *exec.Cmd
	if !testing {
		cmd = exec.CommandContext(ctx,
			"ash", "-c",
			fmt.Sprintf("%s > %s 2>&1 &", strings.Join(args, " "), logFile),
		)
	} else {
		cmd = exec.CommandContext(ctx,
			args[0], args[1:]...,
		)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}
