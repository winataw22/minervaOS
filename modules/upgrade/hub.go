package upgrade

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
)

const (
	// HubBaseURL base hub url
	HubBaseURL = "https://hub.grid.tf/"

	// HubStorage default hub db
	HubStorage = "zdb://hub.grid.tf:9900"
)

// Hub API for f-list
type Hub struct{}

// MountURL returns the full url of given flist.
func (h *Hub) MountURL(flist string) string {
	url, err := url.Parse(HubBaseURL)
	if err != nil {
		panic("invalid base url")
	}
	url.Path = flist
	return url.String()
}

// StorageURL return hub storage url
func (h *Hub) StorageURL() string {
	return HubStorage
}

// Info gets flist info from hub
func (h *Hub) Info(flist string) (info FListInfo, err error) {
	u, err := url.Parse(HubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = filepath.Join("api", "flist", flist, "light")
	response, err := http.Get(u.String())
	if err != nil {
		return info, err
	}

	defer response.Body.Close()
	defer ioutil.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		return info, fmt.Errorf("failed to get flist info: %s", response.Status)
	}

	dec := json.NewDecoder(response.Body)

	err = dec.Decode(&info)
	return info, err
}

// LoadInfo get boot info set by bootstrap process
func LoadInfo(path string) (info FListInfo, err error) {
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}

	defer f.Close()
	dec := json.NewDecoder(f)

	err = dec.Decode(&info)
	return info, err
}

// FListInfo reflects node boot information (flist + version)
type FListInfo struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	Type    string `json:"type"`
	Updated uint64 `json:"updated"`
	Hash    string `json:"hash"`
	Size    uint64 `json:"size"`
}

// Commit write version to version file
func (b *FListInfo) Commit(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0400)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	return enc.Encode(b)
}

func (b *FListInfo) extractVersion(name string) (ver semver.Version, err error) {
	// name is suppose to be as follows
	// <name>:<version>.flist
	parts := strings.Split(name, ":")
	last := parts[len(parts)-1]
	return semver.Parse(strings.TrimSuffix(last, ".flist"))
}

// Version returns the version of the flist
func (b *FListInfo) Version() (semver.Version, error) {
	// computing the version is tricky because it's part of the name
	// of the flist (or the target) depends on the type of the flist

	if b.Type == "symlink" {
		return b.extractVersion(b.Target)
	}

	return b.extractVersion(b.Name)
}
