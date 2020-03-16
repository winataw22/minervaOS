package provision

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	generated "github.com/threefoldtech/zos/pkg/gedis/types/provision"
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
func (s *HTTPStore) Reserve(r *Reservation) (string, error) {
	if r.NodeID == "" {
		return "", fmt.Errorf("nodeID cannot be empty in the reservation")
	}
	url := fmt.Sprintf("%s/reservations/%s", s.baseURL, r.NodeID)

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

// Poll retrieves reservations from BCDB. from acts like a cursor, first call should use
// 0  to retrieve everything. Next calls should use the last+1 ID of the previous poll.
// Note that from is a reservation ID not a workload ID. so user the Reservation.SplitID() method
// to get the reservation part.
func (s *HTTPStore) Poll(nodeID pkg.Identifier, from uint64) ([]*Reservation, error) {
	u, err := url.Parse(fmt.Sprintf("%s/reservations/workloads/%s", s.baseURL, nodeID.Identity()))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Add("from", fmt.Sprintf("%d", from))

	u.RawQuery = q.Encode()

	log.Debug().Str("url", u.String()).Msg("fetching")

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, NewErrTemporary(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewErrTemporary(err)
	}

	if resp.Header.Get("content-type") != "application/json" {
		return nil, NewErrTemporary(err)
	}

	workloads := []generated.TfgridReservationWorkload1{}
	if err := json.NewDecoder(resp.Body).Decode(&workloads); err != nil {
		return nil, err
	}
	reservations := make([]*Reservation, 0, len(workloads))
	for _, wl := range workloads {
		r, err := WorkloadToProvisionType(wl)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load workload type")
		}
		r.Tag = Tag{"source": "HTTPStore"}
		reservations = append(reservations, r)
	}
	return reservations, nil
}

// Get retrieves a single reservation using its ID
func (s *HTTPStore) Get(id string) (*Reservation, error) {
	url := fmt.Sprintf("%s/reservations/workloads/%s", s.baseURL, id)

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
	r.Tag = Tag{"source": "HTTPSource"}
	return r, nil
}

// Feedback sends back the result of a provisioning to BCDB
func (s *HTTPStore) Feedback(nodeID string, r *Result) error {
	url := fmt.Sprintf("%s/reservations/workloads/%s/%s", s.baseURL, r.ID, nodeID)

	buf := &bytes.Buffer{}

	if err := json.NewEncoder(buf).Encode(r); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, url, buf)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status code %s", resp.Status)
	}
	return nil
}

// Deleted marks a reservation as deleted
func (s *HTTPStore) Deleted(nodeID, id string) error {
	url := fmt.Sprintf("%s/reservations/workloads/%s/%s", s.baseURL, id, nodeID)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
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

// Delete marks a reservation as to be deleted, signature
// is of the `str(reservation.id) + reservation.json`
func (s *HTTPStore) Delete(userID, resID int64, sig []byte) error {
	url := fmt.Sprintf("%s/reservations/%d/sign/delete", s.baseURL, resID)

	signature := generated.TfgridReservationSigningSignature1{
		Tid:       userID,
		Signature: hex.EncodeToString(sig),
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(signature); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status code %s", resp.Status)
	}

	return nil
}

// UpdateReservedResources send the amount of resource units reserved to BCDB
func (s *HTTPStore) UpdateReservedResources(nodeID string, c Counters) error {
	url := fmt.Sprintf("%s/nodes/%s/used_resources", s.baseURL, nodeID)

	u := resourceUnits{
		CRU: int64(c.CRU),
		MRU: int64(c.MRU),
		SRU: int64(c.SRU),
		HRU: int64(c.HRU),
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(u); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, buf)
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
