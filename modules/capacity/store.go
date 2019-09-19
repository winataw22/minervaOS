package capacity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/threefoldtech/zosv2/modules/capacity/dmi"

	"github.com/threefoldtech/zosv2/modules"
)

// HTTPStore implement the method to push capacity information to BCDB over HTTP
type HTTPStore struct {
	baseURL string
}

// NewHTTPStore create a new HTTPStore
func NewHTTPStore(url string) *HTTPStore {
	return &HTTPStore{url}
}

// Register sends the capacity information to BCDB
func (s *HTTPStore) Register(nodeID modules.Identifier, c *Capacity, d *dmi.DMI) error {
	x := struct {
		Capacity *Capacity `json:"capacity,omitempty"`
		DMI      *dmi.DMI  `json:"dmi,omitempty"`
	}{
		Capacity: c,
		DMI:      d,
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
func (s *HTTPStore) Ping(nodeID modules.Identifier) error { return nil }
