package farmer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	contentType = "application/json"
)

var (
	successCodes = []int{http.StatusOK}
)

// Client to farmer bot
type Client struct {
	url  url.URL
	base string
}

// NewClient creates a new instance of client
func NewClient(u string) (*Client, error) {
	ul, err := url.Parse(u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse farmer path")
	}

	path := ul.Path
	ul.Path = ""
	return &Client{
		url:  *ul,
		base: path,
	}, nil
}

func (c *Client) path(rel ...string) string {
	u := c.url
	u.Path = filepath.Join(c.base, filepath.Join(rel...))
	return u.String()
}

func (c *Client) serialize(o interface{}) (*bytes.Buffer, error) {
	var bytes bytes.Buffer
	err := json.NewEncoder(&bytes).Encode(o)
	return &bytes, err
}

func (c *Client) response(r *http.Response, o interface{}, codes ...int) error {
	defer r.Body.Close()
	if len(codes) == 0 {
		codes = successCodes
	}

	in := func(i int, l []int) bool {
		for _, v := range l {
			if v == i {
				return true
			}
		}
		return false
	}

	if !in(r.StatusCode, codes) {
		msg, _ := ioutil.ReadAll(r.Body)
		return fmt.Errorf("invalid response (%s): %s", r.Status, string(msg))
	}

	defer func() {
		ioutil.ReadAll(r.Body)
	}()

	if o != nil {
		return json.NewDecoder(r.Body).Decode(o)
	}

	return nil
}
