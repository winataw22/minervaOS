package environment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultHttpTimeout = 10 * time.Second
)

// Config is configuration set by the organization
type Config struct {
	Yggdrasil struct {
		Peers []string `json:"peers"`
	} `json:"yggdrasil"`
	Mycelium struct {
		Peers []string `json:"peers"`
	} `json:"mycelium"`
	Users struct {
		Authorized []string `json:"authorized"`
	} `json:"users"`
	RolloutUpgrade struct {
		TestFarms     []uint32 `json:"test_farms"`
		SafeToUpgrade bool     `json:"safe_to_upgrade"`
	} `json:"rollout_upgrade"`
}

// Merge, updates current config with cfg merging and override config
// based on some update rules.
func (c *Config) Merge(cfg Config) {
	c.Yggdrasil.Peers = uniqueStr(append(c.Yggdrasil.Peers, cfg.Yggdrasil.Peers...))
	// sort peers for easier comparison
	sort.Strings(c.Yggdrasil.Peers)
}

// GetConfig returns extend config for current run mode
func GetConfig() (base Config, err error) {
	env, err := Get()
	if err != nil {
		return
	}
	return GetConfigForMode(env.RunningMode)
}

// GetConfig returns extend config for specific run mode
func GetConfigForMode(mode RunMode) (Config, error) {
	httpClient := &http.Client{
		Timeout: defaultHttpTimeout,
	}

	return getConfig(mode, baseExtendedURL, httpClient)
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

func getConfig(run RunMode, url string, httpClient *http.Client) (ext Config, err error) {
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	u := url + fmt.Sprintf("%s.json", run)

	response, err := httpClient.Get(u)
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
