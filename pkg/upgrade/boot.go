package upgrade

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/upgrade/hub"
)

const (
	// those values must match the values
	// in the bootstrap process. (bootstrap.sh)

	TagFile = "/tmp/tag.info"

	// deprecated file used to detect boot method. we use this
	// as a fall back in case of an upgrade to a running machine
	OldZosFile = "/tmp/flist.name"
)

var (
	ErrNotBootstrapped = fmt.Errorf("node was not bootstrapped")
)

// BootMethod defines the node boot method
type BootMethod string

const (
	// BootMethodBootstrap booted with bootstrapping.
	// this means that all packages are installed from flist
	BootMethodBootstrap BootMethod = "bootstrap"

	// BootMethodOther booted with other methods
	// only happen during development (VM + overlay)
	BootMethodOther BootMethod = "other"
)

func (b BootMethod) IsBootstrapped() bool {
	return b == BootMethodBootstrap
}

// Boot struct
type Boot struct{}

// DetectBootMethod tries to detect the boot method
// of the node
func (b Boot) DetectBootMethod() BootMethod {
	log.Info().Msg("detecting boot method")

	// deprecated file. but if exists we still
	// need to honor the method
	if _, err := os.Stat(OldZosFile); err == nil {
		// if this file existed so we booted normally with
		return BootMethodBootstrap
	}

	if _, err := os.Stat(TagFile); err != nil {
		return BootMethodOther
	}

	// NOTE: we can add a check to see if the flist
	// in the file is valid, but this means we need
	// to do a call to the hub, hence the detection
	// can be affected by the network state, or the
	// hub state. So we return immediately
	return BootMethodBootstrap
}

// Name always return name of the boot flist. If name file
// does not exist, an empty string is returned
func (b *Boot) RunMode() environment.RunMode {
	env := environment.MustGet()
	return env.RunningMode
}

func (b *Boot) Version() semver.Version {
	current, err := b.Current()
	if err != nil {
		return semver.MustParse("0.0.0")
	}

	last := filepath.Base(current.Target)
	var ver semver.Version
	if strings.HasPrefix(last, "v") {
		ver, err = semver.Parse(strings.TrimPrefix(last, "v"))
		if err == nil {
			return ver
		}
	}

	ver.Pre = append(ver.Pre, semver.PRVersion{VersionStr: last})

	return ver
}

// Current returns current flist information
func (b *Boot) Current() (flist hub.TagLink, err error) {
	f, err := os.Open(TagFile)
	if os.IsNotExist(err) {
		return flist, ErrNotBootstrapped
	} else if err != nil {
		return flist, err
	}

	defer f.Close()

	dec := json.NewDecoder(f)

	if err = dec.Decode(&flist); err != nil {
		return flist, err
	}

	if flist.Type != hub.TypeTagLink {
		return flist, fmt.Errorf("expected current installation info to be a taglink, found '%s'", flist.Type)
	}

	return
}

// Set updates the stored flist info
func (b *Boot) Set(c hub.TagLink) error {
	f, err := os.Create(TagFile)
	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(c)
}
