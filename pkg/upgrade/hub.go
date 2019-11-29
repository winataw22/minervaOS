package upgrade

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
)

const (
	// hubBaseURL base hub url
	hubBaseURL = "https://hub.grid.tf/"

	// hubStorage default hub db
	hubStorage = "zdb://hub.grid.tf:9900"
)

// hubClient API for f-list
type hubClient struct{}

// MountURL returns the full url of given flist.
func (h *hubClient) MountURL(flist string) string {
	url, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}
	url.Path = flist
	return url.String()
}

// StorageURL return hub storage url
func (h *hubClient) StorageURL() string {
	return hubStorage
}

// Info gets flist info from hub
func (h *hubClient) Info(flist string) (info flistInfo, err error) {
	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	info.Repository = filepath.Dir(flist)

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

func (h *hubClient) List(repo string) ([]listFListInfo, error) {
	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = filepath.Join("api", "flist", repo)
	response, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	defer ioutil.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get repository listing: %s", response.Status)
	}

	dec := json.NewDecoder(response.Body)

	var result []listFListInfo
	err = dec.Decode(&result)

	for i := range result {
		result[i].Repository = repo
	}

	return result, err
}

type listFListInfo struct {
	Name       string `json:"name"`
	Target     string `json:"target"`
	Type       string `json:"type"`
	Updated    uint64 `json:"updated"`
	Repository string `json:"-"`
}

// flistInfo reflects node boot information (flist + version)
type flistInfo struct {
	listFListInfo
	Hash string `json:"hash"`
	Size uint64 `json:"size"`
}

// fileInfo is the file of an flist
type fileInfo struct {
	Path string `json:"path"`
	Size uint64 `json:"size"`
}

// Commit write version to version file
func (b *flistInfo) Commit(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0400)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	return enc.Encode(b)
}

func (b *listFListInfo) Fqdn() string {
	return path.Join(b.Repository, b.Name)
}

func (b *listFListInfo) extractVersion(name string) (ver semver.Version, err error) {
	// name is suppose to be as follows
	// <name>:<version>.flist
	parts := strings.Split(name, ":")
	last := parts[len(parts)-1]
	last = strings.TrimPrefix(last, "v")
	last = strings.TrimSuffix(last, ".flist")
	return semver.Parse(last)
}

// Version returns the version of the flist
func (b *listFListInfo) Version() (semver.Version, error) {
	// computing the version is tricky because it's part of the name
	// of the flist (or the target) depends on the type of the flist

	if b.Type == "symlink" {
		return b.extractVersion(b.Target)
	}

	return b.extractVersion(b.Name)
}

// Absolute returns the actual flist name
func (b *listFListInfo) Absolute() string {
	name := b.Name
	if b.Type == "symlink" {
		name = b.Target
	}

	return filepath.Join(b.Repository, name)
}

// Files gets the list of the files of an flist
func (b *listFListInfo) Files() ([]fileInfo, error) {
	flist := b.Absolute()
	if len(flist) == 0 {
		return nil, fmt.Errorf("invalid flist info")
	}

	var content struct {
		Content []fileInfo `json:"content"`
	}

	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = filepath.Join("api", "flist", flist)
	response, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	defer ioutil.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get flist info: %s", response.Status)
	}

	dec := json.NewDecoder(response.Body)

	err = dec.Decode(&content)
	return content.Content, err
}
