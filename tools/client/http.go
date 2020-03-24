package client

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
	httpContentType = "application/json"
)

var (
	successCodes = []int{
		http.StatusOK,
		http.StatusCreated,
	}
)

type httpClient struct {
	u  *url.URL
	cl http.Client
}

func newHTTPClient(raw string) (*httpClient, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid url")
	}

	return &httpClient{u: u}, nil
}

func (c *httpClient) url(p ...string) string {
	b := *c.u
	b.Path = filepath.Join(b.Path, filepath.Join(p...))

	return b.String()
}

func (c *httpClient) process(response *http.Response, output interface{}, expect ...int) error {
	defer response.Body.Close()

	if len(expect) == 0 {
		expect = successCodes
	}

	in := func(i int, l []int) bool {
		for _, x := range l {
			if x == i {
				return true
			}
		}
		return false
	}

	dec := json.NewDecoder(response.Body)
	if !in(response.StatusCode, expect) {
		var output struct {
			E string `json:"error"`
		}

		if err := dec.Decode(&output); err != nil {
			return errors.Wrapf(err, "failed to load error while processing invalid return code of: %s", response.Status)
		}

		return fmt.Errorf("%s: %s", response.Status, output.E)
	}

	if output == nil {
		//discard output
		ioutil.ReadAll(response.Body)
		return nil
	}

	if err := dec.Decode(output); err != nil {
		return errors.Wrap(err, "failed to load output")
	}

	return nil
}

func (c *httpClient) post(u string, input interface{}, output interface{}, expect ...int) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(input); err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}

	response, err := c.cl.Post(u, httpContentType, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	return c.process(response, output, expect...)
}

func (c *httpClient) put(u string, input interface{}, output interface{}, expect ...int) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(input); err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}
	req, err := http.NewRequest(http.MethodPut, u, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}
	response, err := c.cl.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	return c.process(response, output, expect...)
}

func (c *httpClient) get(u string, query url.Values, output interface{}, expect ...int) error {
	if len(query) > 0 {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}

	response, err := c.cl.Get(u)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	return c.process(response, output, expect...)
}

func (c *httpClient) delete(u string, query url.Values, output interface{}, expect ...int) error {
	if len(query) > 0 {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}
	response, err := c.cl.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	return c.process(response, output, expect...)
}
