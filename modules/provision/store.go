package provision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/threefoldtech/zosv2/modules/identity"
)

// ReservationStore represent the interface to implement
// to talk to a reservation store
type ReservationStore interface {
	Reserve(r Reservation, nodeID identity.Identifier) error
	Get(id string) (Reservation, error)
}

type httpStore struct {
	baseURL string
}

// NewhHTTPStore create an a client to a TNoDB reachable over HTTP
func NewhHTTPStore(url string) ReservationStore {
	return &httpStore{baseURL: url}
}

func (s *httpStore) Reserve(r Reservation, nodeID identity.Identifier) error {
	url := fmt.Sprintf("%s/reserve/%s", s.baseURL, nodeID.Identity())

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

func (s *httpStore) Get(id string) (Reservation, error) {
	url := fmt.Sprintf("%s/reserve/%s", s.baseURL, id)

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
