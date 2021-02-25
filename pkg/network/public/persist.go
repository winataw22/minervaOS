package public

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

// ErrNoPublicConfig is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPublicConfig = errors.New("no public interface configured for this node")

// LoadPublicConfig loads public config from file
func LoadPublicConfig(path string) (*pkg.PublicConfig, error) {
	file, err := os.Open(path)
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

// SavePublicConfig stores public config in a file
func SavePublicConfig(path string, cfg *pkg.PublicConfig) error {
	if cfg == nil {
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "couldn't delete config file")
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "failed to create configuration file")
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(cfg); err != nil {
		return errors.Wrap(err, "failed to encode public config")
	}

	return nil
}
