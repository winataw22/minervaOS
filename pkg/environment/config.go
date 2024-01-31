package environment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Config is configuration set by the organization
type Config struct {
	Yggdrasil struct {
		Peers []string `json:"peers"`
	} `json:"yggdrasil"`
	Mycelium struct {
		Peers []string `json:"peers"`
	} `json:"mycelium"`
}

// Merge, updates current config with cfg merging and override config
// based on some update rules.
func (c *Config) Merge(cfg Config) {
	c.Yggdrasil.Peers = uniqueStr(append(c.Yggdrasil.Peers, cfg.Yggdrasil.Peers...))
	// sort peers for easier comparison
	sort.Strings(c.Yggdrasil.Peers)
}

// GetConfig returns extend config for specific run mode
func GetConfig() (base Config, err error) {
	env, err := Get()
	if err != nil {
		return
	}
	base, err = getConfig(env.RunningMode, baseExtendedURL)
	if err != nil {
		return
	}
	if env.ExtendedConfigURL != "" {
		custom, err := getConfig(env.RunningMode, env.ExtendedConfigURL)
		if err != nil {
			log.Error().Err(err).Msg("fetching the config from the env config-url failed")
		}
		base.Merge(custom)
	}

	return base, nil
}

func uniqueStr(slice []string) []string {
	keys := make(map[string]struct{})
	list := slice[:0]
	for _, entry := range slice {
		if _, exists := keys[entry]; !exists {
			keys[entry] = struct{}{}
			list = append(list, entry)
		}
	}
	return list
}

func getConfig(run RunMode, url string) (ext Config, err error) {
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	u := url + fmt.Sprintf("%s.json", run)

	response, err := http.Get(u)
	if err != nil {
		return ext, err
	}

	defer func() {
		_, _ = io.ReadAll(response.Body)
	}()

	if response.StatusCode != http.StatusOK {
		return ext, fmt.Errorf("failed to get extended config: %s", response.Status)
	}

	if err := json.NewDecoder(response.Body).Decode(&ext); err != nil {
		return ext, errors.Wrap(err, "failed to decode extended settings")
	}

	return
}
