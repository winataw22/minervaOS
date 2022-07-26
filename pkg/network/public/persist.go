package public

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

const (
	publicConfigFile = "public-config.json"
	publicExitFile   = "public-exit"
)

var (
	// persistencePath is path to config file.
	persistencePath = ""
)

func SetPersistence(path string) {
	stat, err := os.Stat(path)
	if err != nil || !stat.IsDir() {
		panic("invalid persistence path must be an existing directory")
	}

	persistencePath = path
}

func getPersistencePath(file string) string {
	if persistencePath == "" {
		panic("public config persistence path is not set")
	}
	return filepath.Join(persistencePath, file)
}

// ErrNoPublicConfig is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPublicConfig = errors.New("no public configuration")

// LoadPublicConfig loads public config from file
func LoadPublicConfig() (*pkg.PublicConfig, error) {
	file, err := os.Open(getPersistencePath(publicConfigFile))
	if os.IsNotExist(err) {
		// it's not an error to not have config
		// but we return a nil config
		return nil, ErrNoPublicConfig
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to load public config file")
	}

	defer file.Close()
	var cfg pkg.PublicConfig
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, errors.Wrap(err, "failed to decode public config")
	}

	return &cfg, nil
}

func DeletePublicConfig() error {
	return os.RemoveAll(getPersistencePath(publicConfigFile))
}

// SavePublicConfig stores public config in a file
func SavePublicConfig(cfg pkg.PublicConfig) error {
	file, err := os.Create(getPersistencePath(publicConfigFile))
	if err != nil {
		return errors.Wrap(err, "failed to create configuration file")
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(cfg); err != nil {
		return errors.Wrap(err, "failed to encode public config")
	}

	return nil
}
