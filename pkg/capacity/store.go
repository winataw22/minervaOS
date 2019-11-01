package capacity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/gedis"

	"github.com/threefoldtech/zos/pkg"
)

// Store is an interface to the bcdb store to report capacity
type Store interface {
	Register(nodeID pkg.Identifier, c Capacity, d dmi.DMI, disks Disks) error
	Ping(nodeID pkg.Identifier, uptime uint64) error
}

// BCDBStore implements the store interface using a gedis client to BCDB
type BCDBStore struct {
	g *gedis.Gedis
}

// NewBCDBStore creates a BCDBStore
func NewBCDBStore(gedis *gedis.Gedis) *BCDBStore {
	return &BCDBStore{g: gedis}
}

// Register sends the capacity information to BCDB
func (s *BCDBStore) Register(nodeID pkg.Identifier, c Capacity, d dmi.DMI, disks Disks) error {
	if err := s.g.UpdateTotalNodeCapacity(nodeID, c.MRU, c.CRU, c.HRU, c.SRU); err != nil {
		return err
	}

	return s.g.SendHardwareProof(nodeID, d, disks)
}

// Ping sends an heart-beat to BCDB
func (s *BCDBStore) Ping(nodeID pkg.Identifier, uptime uint64) error {
	return s.g.UptimeUpdate(nodeID, uptime)
}

// HTTPStore implement the method to push capacity information to BCDB over HTTP
type HTTPStore struct {
	baseURL string
}

// NewHTTPStore create a new HTTPStore
func NewHTTPStore(url string) *HTTPStore {
	return &HTTPStore{url}
}

// Register sends the capacity information to BCDB
func (s *HTTPStore) Register(nodeID pkg.Identifier, c Capacity, d dmi.DMI, disks Disks) error {
	x := struct {
		Capacity Capacity `json:"capacity"`
		DMI      dmi.DMI  `json:"dmi"`
		Disks    Disks    `json:"disks"`
	}{
		Capacity: c,
		DMI:      d,
		Disks:    disks,
	}
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(x)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(s.baseURL+"/nodes/%s/capacity", nodeID.Identity())
	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	return nil
}

// Ping sends an heart-beat to BCDB
func (s *HTTPStore) Ping(nodeID pkg.Identifier, uptime uint64) error {
	x := struct {
		Uptime uint64 `json:"uptime"`
	}{uptime}

	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(x)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(s.baseURL+"/nodes/%s/uptime", nodeID.Identity())
	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	return nil
}
