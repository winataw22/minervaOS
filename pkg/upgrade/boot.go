package upgrade

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	// those values must match the values
	// in the bootstrap process. (bootstrap.sh)

	nameFile = "/tmp/flist.name"
	infoFile = "/tmp/flist.info"
	binsFile = "/tmp/bins.info"
)

// BootMethod defines the node boot method
type BootMethod string

const (
	// BootMethodFList booted from an flist
	BootMethodFList BootMethod = "flist"

	// BootMethodOther booted with other methods
	BootMethodOther BootMethod = "other"
)

//Boot struct
type Boot struct{}

// DetectBootMethod tries to detect the boot method
// of the node
func (b Boot) DetectBootMethod() BootMethod {
	log.Info().Msg("detecting boot method")
	_, err := os.Stat(nameFile)
	if err != nil {
		log.Warn().Err(err).Msg("no flist file found")
		return BootMethodOther
	}

	// NOTE: we can add a check to see if the flist
	// in the file is valid, but this means we need
	// to do a call to the hub, hence the detection
	// can be affected by the network state, or the
	// hub state. So we return immediately
	return BootMethodFList
}

// Name always return name of the boot flist. If name file
// does not exist, an empty string is returned
func (b *Boot) Name() string {
	data, _ := ioutil.ReadFile(nameFile)
	return strings.TrimSpace(string(data))
}

//CurrentBins returns a list of current binaries installed
func (b *Boot) CurrentBins() (map[string]RepoFList, error) {
	f, err := os.Open(binsFile)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(f)

	var result map[string]RepoFList
	err = dec.Decode(&result)
	return result, err
}

//SetBins sets the current list of binaries in boot files
func (b *Boot) SetBins(current map[string]RepoFList) error {
	f, err := os.Create(binsFile)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	return enc.Encode(current)
}

// Current returns current flist information
func (b *Boot) Current() (FListEvent, error) {
	name := b.Name()
	if len(name) == 0 {
		return FListEvent{}, fmt.Errorf("flist name is not known")
	}

	info, err := loadInfo(name, infoFile)
	if err != nil {
		return FListEvent{}, err
	}

	return FListEvent{info}, nil
}

// Set updates the stored flist info
func (b *Boot) Set(c FListEvent) error {
	return c.Commit(infoFile)
}

// Version always returns curent version of flist
func (b *Boot) Version() (semver.Version, error) {
	info, err := b.Current()
	if err != nil {
		return semver.Version{}, errors.Wrap(err, "failed to load flist info")
	}

	return info.Version()
}

//MustVersion must returns the current version or panic
func (b *Boot) MustVersion() semver.Version {
	ver, err := b.Version()
	if err != nil {
		panic(err)
	}

	return ver
}

// loadInfo get boot info set by bootstrap process
func loadInfo(fqn string, path string) (info flistInfo, err error) {
	info.Repository = filepath.Dir(fqn)
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}

	defer f.Close()
	dec := json.NewDecoder(f)

	err = dec.Decode(&info)
	return info, err
}
