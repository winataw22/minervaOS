package provision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/threefoldtech/zosv2/modules"
)

// HTTPStore is a reservation store
// over HTTP
type HTTPStore struct {
	baseURL string
}

// NewHTTPStore creates an a client to a TNoDB reachable over HTTP
func NewHTTPStore(url string) *HTTPStore {
	return &HTTPStore{baseURL: url}
}

// Reserve adds a reservation to the BCDB
func (s *HTTPStore) Reserve(r *Reservation, nodeID modules.Identifier) error {
	url := fmt.Sprintf("%s/reservations/%s", s.baseURL, nodeID.Identity())

	buf := &bytes.Buffer{}

	if err := json.NewEncoder(buf).Encode(r); err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status code %s", resp.Status)
	}

	return nil
}

// Poll retrieves reservations from BCDB. If all is true, it returns all the reservations
// for this node, otherwise is return only the reservation never sent yet and do long polling
func (s *HTTPStore) Poll(nodeID modules.Identifier, all bool) ([]*Reservation, error) {
	u, err := url.Parse(fmt.Sprintf("%s/reservations/%s/poll", s.baseURL, nodeID.Identity()))
	if err != nil {
		return nil, err
	}
	if all {
		q := u.Query()
		q.Add("all", "true")
		u.RawQuery = q.Encode()
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reservation request returned: %s", resp.Status)
	}

	if resp.Header.Get("content-type") != "application/json" {
		return nil, fmt.Errorf("reservation request returned '%s', expected 'application/json'", resp.Header.Get("content-type"))
	}

	reservations := []*Reservation{}
	if err := json.NewDecoder(resp.Body).Decode(&reservations); err != nil {
		return nil, err
	}
	return reservations, nil
}

// Get retrieves a single reservation using its ID
func (s *HTTPStore) Get(id string) (*Reservation, error) {
	url := fmt.Sprintf("%s/reservations/%s", s.baseURL, id)

	r := &Reservation{}
	resp, err := http.Get(url)
	if err != nil {
		return r, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return r, fmt.Errorf("wrong response status code %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
		return r, err
	}

	return r, nil
}
