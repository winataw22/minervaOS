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
	"github.com/zaibon/httpsig"
)

var (
	successCodes = []int{
		http.StatusOK,
		http.StatusCreated,
	}
)

type httpClient struct {
	u        *url.URL
	cl       http.Client
	signer   *httpsig.Signer
	identity string
}

// HTTPError is the error type returned by the client
// it contains the error and the HTTP response
type HTTPError struct {
	resp *http.Response
	err  error
}

func (h HTTPError) Error() string {
	return fmt.Sprintf("%v status:%s", h.err, h.resp.Status)
}

// Response return the HTTP response that trigger this error
func (h HTTPError) Response() http.Response {
	return *h.resp
}

func newHTTPClient(raw string, id Identity) (*httpClient, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid url")
	}

	var signer *httpsig.Signer
	if id != nil {
		signer = httpsig.NewSigner(id.Identity(), id.PrivateKey(), httpsig.Ed25519, []string{"(created)", "date", "threebot-id"})
	}

	return &httpClient{
		u:        u,
		signer:   signer,
		identity: id.Identity(),
	}, nil
}

func (c *httpClient) url(p ...string) string {
	b := *c.u
	b.Path = filepath.Join(b.Path, filepath.Join(p...))

	return b.String()
}

func (c *httpClient) sign(r *http.Request) error {
	if c.signer == nil {
		return nil
	}

	r.Header.Set(http.CanonicalHeaderKey("threebot-id"), c.identity)
	return c.signer.Sign(r)
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
			return errors.Wrapf(HTTPError{
				err:  err,
				resp: response,
			}, "failed to load error while processing invalid return code of: %s", response.Status)
		}

		return HTTPError{
			err:  fmt.Errorf(output.E),
			resp: response,
		}
	}

	if output == nil {
		//discard output
		ioutil.ReadAll(response.Body)
		return nil
	}

	if err := dec.Decode(output); err != nil {
		return HTTPError{
			err:  errors.Wrap(err, "failed to load output"),
			resp: response,
		}
	}

	return nil
}

func (c *httpClient) post(u string, input interface{}, output interface{}, expect ...int) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(input); err != nil {
		return errors.Wrap(err, "failed to serialize request body")
	}

	req, err := http.NewRequest(http.MethodPost, u, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to create new HTTP request")
	}

	if err := c.sign(req); err != nil {
		return errors.Wrap(err, "failed to sign HTTP request")
	}
	response, err := c.cl.Do(req)
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

	if err := c.sign(req); err != nil {
		return errors.Wrap(err, "failed to sign HTTP request")
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

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create new HTTP request")
	}

	if err := c.sign(req); err != nil {
		return errors.Wrap(err, "failed to sign HTTP request")
	}

	response, err := c.cl.Do(req)
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

	if err := c.sign(req); err != nil {
		return errors.Wrap(err, "failed to sign HTTP request")
	}

	response, err := c.cl.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}

	return c.process(response, output, expect...)
}
