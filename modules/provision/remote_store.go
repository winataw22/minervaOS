package provision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
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
func (s *HTTPStore) Reserve(r *Reservation, nodeID modules.Identifier) (string, error) {
	url := fmt.Sprintf("%s/reservations/%s", s.baseURL, nodeID.Identity())

	buf := &bytes.Buffer{}

	if err := json.NewEncoder(buf).Encode(r); err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Extract the Location header which contains
	// url to get information about the created resource
	resource := resp.Header.Get("Location")

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("wrong response status code %s", resp.Status)
	}

	return resource, nil
}

// Poll retrieves reservations from BCDB. If all is true, it returns all the reservations
// for this node.
// otherwise it returns only the reservation never sent yet or the reservation that needs to be deleted
// and do long polling
func (s *HTTPStore) Poll(nodeID modules.Identifier, all bool, since time.Time) ([]*Reservation, error) {
	u, err := url.Parse(fmt.Sprintf("%s/reservations/%s/poll", s.baseURL, nodeID.Identity()))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if all {
		q.Add("all", "true")
	}
	if since.Unix() > 0 {
		q.Add("since", fmt.Sprintf("%d", since.Unix()))
	}
	u.RawQuery = q.Encode()

	log.Info().Str("url", u.String()).Msg("fetching")

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

// Feedback sends back the result of a provisioning to BCDB
func (s *HTTPStore) Feedback(id string, r *Result) error {
	url := fmt.Sprintf("%s/reservations/%s", s.baseURL, id)

	buf := &bytes.Buffer{}

	if err := json.NewEncoder(buf).Encode(r); err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, buf)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong response status code %s", resp.Status)
	}
	return nil
}

// Deleted marks a reservation as deleted
func (s *HTTPStore) Deleted(id string) error {
	url := fmt.Sprintf("%s/reservations/%s/deleted", s.baseURL, id)

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong response status code %s", resp.Status)
	}
	return nil
}
