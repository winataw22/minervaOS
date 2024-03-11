package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/0-fs/meta"
)

const (
	// hubBaseURL base hub url
	hubBaseURL = "https://hub.grid.tf/"

	// hubStorage default hub db
	hubStorage = "zdb://hub.grid.tf:9900"

	defaultHubCallTimeout = 20 * time.Second
)

type FListType string

const (
	TypeRegular FListType = "regular"
	TypeSymlink FListType = "symlink"
	TypeTag     FListType = "tag"
	TypeTagLink FListType = "taglink"
)

type FListFilter interface {
	matches(*FList) bool
}

type matchName struct {
	name string
}

func (m matchName) matches(f *FList) bool {
	return f.Name == m.name
}

type matchType struct {
	typ FListType
}

func (m matchType) matches(f *FList) bool {
	return f.Type == m.typ
}

func MatchName(name string) FListFilter {
	return matchName{name}
}

func MatchType(typ FListType) FListFilter {
	return matchType{typ}
}

// HubClient API for f-list
type HubClient struct {
	httpClient *http.Client
}

// NewHubClient create new hub client with the passed option for the http client
func NewHubClient(timeout time.Duration) *HubClient {
	return &HubClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

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
func (h *HubClient) Info(repo, name string) (info FList, err error) {
	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = filepath.Join("api", "flist", repo, name, "light")

	response, err := h.httpClient.Get(u.String())
	if err != nil {
		return info, err
	}

	defer response.Body.Close()
	defer func() {
		_, _ = io.ReadAll(response.Body)
	}()

	if response.StatusCode != http.StatusOK {
		return info, fmt.Errorf("failed to get flist (%s/%s) info: %s", repo, name, response.Status)
	}

	dec := json.NewDecoder(response.Body)

	err = dec.Decode(&info)
	return info, err
}

func (h *HubClient) Find(repo string, filter ...FListFilter) ([]FList, error) {
	result, err := h.List(repo)
	if err != nil {
		return nil, err
	}

	filtered := result[:0]
next:
	for _, flist := range result {
		for _, m := range filter {
			if !m.matches(&flist) {
				continue next
			}
		}
		filtered = append(filtered, flist)
	}

	return filtered, nil
}

// List list repo flists
func (h *HubClient) List(repo string) ([]FList, error) {
	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = filepath.Join("api", "flist", repo)

	response, err := h.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	defer func() {
		_, _ = io.ReadAll(response.Body)
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get repository listing: %s", response.Status)
	}

	dec := json.NewDecoder(response.Body)

	var result []FList
	err = dec.Decode(&result)

	return result, err
}

func (h *HubClient) ListTag(repo, tag string) ([]Symlink, error) {
	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = filepath.Join("api", "flist", repo, "tags", tag)

	response, err := h.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	defer func() {
		_, _ = io.ReadAll(response.Body)
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get repository listing: %s", response.Status)
	}

	dec := json.NewDecoder(response.Body)

	var result []Symlink
	err = dec.Decode(&result)

	return result, err
}

// Download downloads an flist  to cache and return the full
// path to the extraced meta data directory. the returned path is in format
// {cache}/{hash}/
func (h *HubClient) Download(cache, repo, name string) (string, error) {
	log := log.With().Str("cache", cache).Str("repo", repo).Str("name", name).Logger()

	log.Info().Msg("attempt downloading flist")

	info, err := h.Info(repo, name)
	if err != nil {
		return "", err
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

	if stat, err := os.Stat(filepath.Join(extracted, dbFileName)); err == nil {
		// already exists.
		if stat.Size() > 0 {
			log.Info().Msg("already cached")
			return extracted, nil
		}
	}

	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = filepath.Join(repo, name)
	log.Debug().Str("url", u.String()).Msg("downloading flist")

	response, err := h.httpClient.Get(u.String())
	if err != nil {
		return "", errors.Wrap(err, "failed to download flist")
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download flist: %s", response.Status)
	}

	return extracted, meta.Unpack(response.Body, extracted)
}

// FList is information of flist as returned by repo list operation
type FList struct {
	Name    string    `json:"name"`
	Target  string    `json:"target"`
	Type    FListType `json:"type"`
	Updated uint64    `json:"updated"`
	Hash    string    `json:"md5"`
}

// FileInfo is the file of an flist
type FileInfo struct {
	Path string `json:"path"`
	Size uint64 `json:"size"`
}

type Regular struct {
	FList
}

func NewRegular(flist FList) Regular {
	if flist.Type != TypeRegular {
		panic("invalid flist type")
	}

	return Regular{flist}
}

// Files gets the list of the files of an flist
func (b *Regular) Files(repo string) ([]FileInfo, error) {
	var content struct {
		Content []FileInfo `json:"content"`
	}

	u, err := url.Parse(hubBaseURL)
	if err != nil {
		panic("invalid base url")
	}

	u.Path = filepath.Join("api", "flist", repo, b.Name)
	cl := &http.Client{
		Timeout: defaultHubCallTimeout,
	}

	response, err := cl.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	defer func() {
		_, _ = io.ReadAll(response.Body)
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get flist info: %s", response.Status)
	}

	dec := json.NewDecoder(response.Body)

	err = dec.Decode(&content)
	return content.Content, err
}

// TagLink is an flist of type taglink
type TagLink struct {
	FList
}

func NewTagLink(flist FList) TagLink {
	if flist.Type != TypeTagLink {
		panic("invalid flist type")
	}

	return TagLink{flist}
}

func (t *TagLink) Destination() (repo string, tag string, err error) {
	parts := strings.Split(t.Target, "/")
	if len(parts) != 3 || parts[1] != "tags" {
		return repo, tag, fmt.Errorf("invalid target '%s' for taglink", t.Target)
	}

	return parts[0], parts[2], nil
}

type Symlink struct {
	FList
}

func NewSymlink(flist FList) Symlink {
	if flist.Type != TypeSymlink {
		panic("invalid flist type")
	}

	return Symlink{flist}
}

// Destination gets destination flist for a symlink flist
// source repo is the repo where this symlink lives, since the symlink
// can either be an absolute or relative target.
func (t *Symlink) Destination(source string) (repo string, name string, err error) {
	parts := strings.Split(t.Target, "/")
	if len(parts) == 1 {
		return source, t.Target, nil
	} else if len(parts) == 2 {
		return parts[0], parts[1], nil
	} else {
		return repo, name, fmt.Errorf("invalid target '%s' for symlink", t.Target)
	}
}
