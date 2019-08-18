package provision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/threefoldtech/zosv2/modules"
)

// ReservationStore represent the interface to implement
// to talk to a reservation store
type ReservationStore interface {
	// Reserve adds a reservation to the store, the reservation will be eventually
	// picked up by a node to provision workloads
	Reserve(r Reservation, nodeID modules.Identifier) error
	// Poll ask the store to send us reservation for a specific node ID
	// if all is true, the store sends all the reservation every registered for the node ID
	// otherwise it just sends reservation not pulled yet.
	Poll(nodeID modules.Identifier, all bool) ([]*Reservation, error)
	// Get retrieve a specific reservation from the store
	Get(id string) (Reservation, error)
}

type httpStore struct {
	baseURL string
}

// NewhHTTPStore create an a client to a TNoDB reachable over HTTP
func NewhHTTPStore(url string) ReservationStore {
	return &httpStore{baseURL: url}
}

func (s *httpStore) Reserve(r Reservation, nodeID modules.Identifier) error {
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

func (s *httpStore) Poll(nodeID modules.Identifier, all bool) ([]*Reservation, error) {
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

func (s *httpStore) Get(id string) (Reservation, error) {
	url := fmt.Sprintf("%s/reservations/%s", s.baseURL, id)

	r := Reservation{}
	resp, err := http.Get(url)
	if err != nil {
		return r, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return r, fmt.Errorf("wrong response status code %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return r, err
	}

	return r, nil
}
