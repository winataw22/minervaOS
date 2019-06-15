package upgrade

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/blang/semver"
	"github.com/rs/zerolog/log"
)

// Upgrade represent an flist that contains new binaries and lib
// for 0-OS
type Upgrade struct {
	Flist         string `json:"flist"`             // url of the upgrade flist
	Signature     string `json:"signature"`         // signature of the upgrade flist
	TransactionID string `json:"transaction_id"`    //id of the upgrade transaction
	Storage       string `json:"storage,omitempty"` //url of the 0-db used to store the flist data
}

// Publisher is the interface that define how the upgrade are published
type Publisher interface {
	// Get retrieve the Upgrade object for a specific version
	Get(version semver.Version) (Upgrade, error)
	//Latest return the latest version available
	Latest() (semver.Version, error)
	// List all the version this publisher has
	List() ([]semver.Version, error)
}

type httpPublisher struct {
	url string
}

var _ Publisher = (*httpPublisher)(nil)

// NewHTTPPublisher returns a Publisher that uses an HTTP server as a source of upgrade
func NewHTTPPublisher(url string) Publisher {
	return &httpPublisher{
		url: url,
	}
}

func (p *httpPublisher) Latest() (semver.Version, error) {
	var version semver.Version
	url, err := joinURL(p.url, "latest")
	if err != nil {
		return version, err
	}

	log.Info().Str("url", url).Msg("check latest version")

	resp, err := http.Get(url)
	if err != nil {
		log.Error().Str("url", url).Err(err).Msg("fail to get latest version from server")
		return version, err
	}
	if resp.StatusCode != http.StatusOK {
		return version, fmt.Errorf("fail to get latest version from server")
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		log.Error().Err(err).Msg("fail to decode json")
		return version, err
	}

	return version, nil
}

func (p *httpPublisher) List() ([]semver.Version, error) {
	var versions []semver.Version
	url, err := joinURL(p.url, "versions")
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Error().Str("url", url).Err(err).Msg("fail to list versions from publisher")
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fail to list versions from publisher")
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		log.Error().Err(err).Msg("fail to decode json")
		return nil, err
	}

	return versions, nil
}

func (p *httpPublisher) Get(version semver.Version) (Upgrade, error) {
	upgrade := Upgrade{}

	url, err := joinURL(p.url, version.String())
	if err != nil {
		return upgrade, err
	}
	log.Info().Str("url", url).Msg("check for upgrade")

	resp, err := http.Get(url)
	if err != nil {
		log.Error().Err(err).Msg("fail to get upgrade from publisher")
		return upgrade, err
	}
	if resp.StatusCode != http.StatusOK {
		return upgrade, fmt.Errorf("fail to get upgrade from publisher")
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&upgrade); err != nil {
		log.Error().Err(err).Msg("fail to decode json")
		return upgrade, err
	}

	log.Info().Msg("upgrade found")
	return upgrade, nil
}

func joinURL(base, path string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = filepath.Join(u.Path, path)
	return u.String(), nil
}
