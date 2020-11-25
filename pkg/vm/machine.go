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

const (
	configFileName = "config.json"
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

// Jailed represents a jailed machine.
type Jailed struct {
	Machine
	Root string `json:"-"`
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

// Jail will move files to module base and returned a jailed machine.
func (m *Machine) Jail(base string) (*Jailed, error) {
	root := m.root(base)
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, errors.Wrapf(err, "failed to create machine root '%s'", m.ID)
	}

	cfg := *m

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
			return nil, errors.Wrapf(err, "failed to bind mount '%s' to machine root", file)
		}

		*str = filepath.Base(file)
	}

	// mount drives
	for i, drive := range cfg.Drives {
		if len(drive.Path) == 0 {
			return nil, fmt.Errorf("invalid configured disk empty path '%s'", m.ID)
		}

		name := filepath.Base(drive.Path)
		if err := mount(drive.Path, rooted(drive.Path)); err != nil {
			return nil, errors.Wrapf(err, "failed to bind mount '%s' to machine root", drive.Path)
		}

		m.Drives[i].Path = name
	}

	return &Jailed{Machine: cfg, Root: root}, nil
}

// JailedFromPath loads a jailed machine from given path.
// the root points to directory which has `config.json` from
// a previous Save() call
func JailedFromPath(root string) (*Jailed, error) {
	cfg, err := os.Open(filepath.Join(root, configFileName))
	if err != nil {
		return nil, err
	}
	defer cfg.Close()
	var j Jailed
	dec := json.NewDecoder(cfg)
	if err := dec.Decode(&j); err != nil {
		return nil, err
	}
	// extract ID from root
	// root is always in format <base>/firecracker/<id>/root
	j.ID = filepath.Base(filepath.Dir(root))
	j.Root = root
	return &j, nil
}

func (j *Jailed) base() string {
	t := filepath.Join("/firecracker", j.ID, "root")
	return filepath.Clean(strings.TrimSuffix(j.Root, t))
}

// Save configuration
func (j *Jailed) Save() error {
	cfg, err := os.Create(filepath.Join(j.Root, "config.json"))
	if err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	defer cfg.Close()
	enc := json.NewEncoder(cfg)
	if err := enc.Encode(j); err != nil {
		return err
	}

	return cfg.Close()

}

// Start starts the machine.
func (j *Jailed) Start(ctx context.Context) error {
	return j.exec(ctx)
}

// Log returns machine log file path
func (j *Jailed) Log(base string) string {
	return filepath.Join(j.Root, "machine.log")
}

func (j *Jailed) exec(ctx context.Context) error {
	// prepare command
	// because the --daemonize flag does not work as expected
	// we are daemonizing with `ash and &` so we can use cmd.Run().
	// the reason we don't want to use 'cmd.Start' instead of 'cmd.Run'
	// is that if we do 'Start' and then the process exited for some reason
	// we will endup with 'zombi' process because vmd did not 'wait' for the
	// process exit. hence daemonizing is required.
	// due to issues with jailer --daemonize flag, we use the ash trick below

	for _, d := range []string{"dev", "run", "api.socket"} {
		err := os.RemoveAll(filepath.Join(j.Root, d))
		if err == nil || os.IsNotExist(err) {
			continue
		} else {
			return errors.Wrap(err, "failed to cleanup machine root")
		}
	}

	base := j.base() // that is module root

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
		"--id", j.ID,
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

	logFile := j.Log(base)

	var cmd *exec.Cmd
	// okay we use ash as a way to daemonize the firecracker process
	// for somereason doing a cmd.Start() only will make the process
	// killed if the vmd exits! which is a weird behavior not like
	// what is expected (we also tried process.Release()) and also
	// tried to use syscall.SysProcAttr with no luck.
	// TODO:clean up hack use go-daemon
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
