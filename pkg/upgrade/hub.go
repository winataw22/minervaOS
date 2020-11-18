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
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/0-fs/meta"
)

const (
	// hubBaseURL base hub url
	hubBaseURL = "https://hub.grid.tf/"

	// hubStorage default hub db
	hubStorage = "zdb://hub.grid.tf:9900"
)

// HubClient API for f-list
type HubClient struct{}

// MountURL returns the full url of given flist.
func (h *HubClient) MountURL(flist string) string {
	url, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}
	url.Path = flist
	return url.String()
}

// StorageURL return hub storage url
func (h *HubClient) StorageURL() string {
	return hubStorage
}

// Info gets flist info from hub
func (h *HubClient) Info(flist string) (info FullFListInfo, err error) {
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
		return info, fmt.Errorf("failed to get flist (%s) info: %s", flist, response.Status)
	}
	defer response.Body.Close()

	dec := json.NewDecoder(response.Body)

	err = dec.Decode(&info)
	return info, err
}

// List list repo flists
func (h *HubClient) List(repo string) ([]FListInfo, error) {
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

	var result []FListInfo
	err = dec.Decode(&result)

	for i := range result {
		result[i].Repository = repo
	}

	return result, err
}

// Download downloads an flist (fqn: repo/name) to cache and return the full
// path to the extraced meta data directory. the returned path is in format
// {cache}/{hash}/
func (h *HubClient) Download(cache, flist string) (string, error) {
	var info FullFListInfo
	for {
		var err error
		info, err = h.Info(flist)
		if err != nil {
			return "", err
		}
		if info.Type == "symlink" {
			flist = filepath.Join(filepath.Dir(flist), info.Target)
		} else if info.Type == "regular" {
			break
		} else {
			return "", fmt.Errorf("unknown flist type: %s", info.Type)
		}
	}

	if info.Hash == "" {
		return "", fmt.Errorf("invalid flist info returned")
	}

	const (
		dbFileName = "flistdb.sqlite3"
	)

	// check if already downloaded
	downloaded := filepath.Join(cache, info.Hash)
	extracted := fmt.Sprintf("%s.d", downloaded)

	if _, err := os.Stat(filepath.Join(extracted, dbFileName)); err == nil {
		// already exists.
		return extracted, nil
	}

	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = flist
	log.Debug().Str("url", u.String()).Msg("downloading flist")
	response, err := http.Get(u.String())
	if err != nil {
		return "", errors.Wrap(err, "failed to download flist")
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download flist: %s", response.Status)
	}

	return extracted, meta.Unpack(response.Body, extracted)
}

// FListInfo is information of flist as returned by repo list operation
type FListInfo struct {
	Name       string `json:"name"`
	Target     string `json:"target"`
	Type       string `json:"type"`
	Updated    uint64 `json:"updated"`
	Repository string `json:"-"`
}

// FullFListInfo reflects node boot information (flist + version)
type FullFListInfo struct {
	FListInfo
	Hash string `json:"md5"`
	Size uint64 `json:"size"`
}

// FileInfo is the file of an flist
type FileInfo struct {
	Path string `json:"path"`
	Size uint64 `json:"size"`
}

// Commit write version to version file
func (b *FullFListInfo) Commit(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0400)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	return enc.Encode(b)
}

// Fqdn return the full flist name
func (b *FListInfo) Fqdn() string {
	return path.Join(b.Repository, b.Name)
}

func (b *FListInfo) extractVersion(name string) (ver semver.Version, err error) {
	// name is suppose to be as follows
	// <name>:<version>.flist
	parts := strings.Split(name, ":")
	last := parts[len(parts)-1]
	last = strings.TrimPrefix(last, "v")
	last = strings.TrimSuffix(last, ".flist")
	return semver.Parse(last)
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

// Absolute returns the actual flist name
func (b *FListInfo) Absolute() string {
	name := b.Name
	if b.Type == "symlink" {
		name = b.Target
	}

	return filepath.Join(b.Repository, name)
}

// Files gets the list of the files of an flist
func (b *FListInfo) Files() ([]FileInfo, error) {
	flist := b.Absolute()
	if len(flist) == 0 {
		return nil, fmt.Errorf("invalid flist info")
	}

	var content struct {
		Content []FileInfo `json:"content"`
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
