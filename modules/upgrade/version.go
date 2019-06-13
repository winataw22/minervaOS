package upgrade

import (
	"encoding/json"
	"os"

	"github.com/blang/semver"
	"github.com/rs/zerolog/log"
)

//writeVersion write version to path in an atomic way
func writeVersion(path string, version semver.Version) error {
	tmp := path + ".tmp"
	// the file doesn't exist yet. So we are on a fresh system
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0660)
	if err != nil {
		log.Error().Err(err).Msg("open")
		return err
	}

	if err := json.NewEncoder(f).Encode(version); err != nil {
		log.Error().Err(err).Msg("encode")
		f.Close()
		return err
	}

	if err := f.Close(); err != nil {
		log.Error().Err(err).Msg("close")
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		log.Error().Err(err).Msg("rename")
		return err
	}
	return nil
}

func readVersion(path string) (semver.Version, error) {
	var version semver.Version

	f, err := os.Open(path)
	if err != nil {
		return version, err
	}

	// version file exist, just read from it
	if err := json.NewDecoder(f).Decode(&version); err != nil {
		return version, err
	}
	return version, nil
}
