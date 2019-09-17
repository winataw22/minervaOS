package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/threefoldtech/zosv2/modules"
)

// IDStore is the interface defining the
// client side of an identity store
type IDStore interface {
	RegisterNode(node modules.Identifier, farm modules.Identifier, version string) (string, error)
	RegisterFarm(farm modules.Identifier, name string, email string, wallet []string) (string, error)
}

type httpIDStore struct {
	baseURL string
}

// NewHTTPIDStore returns a HTTP IDStore client
func NewHTTPIDStore(url string) IDStore {
	return &httpIDStore{baseURL: url}
}

type nodeRegisterReq struct {
	NodeID string `json:"node_id"`
	FarmID string `json:"farm_id"`
}

// RegisterNode implements the IDStore interface
func (s *httpIDStore) RegisterNode(node modules.Identifier, farm modules.Identifier, version string) (string, error) {
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(nodeRegisterReq{
		NodeID: node.Identity(),
		FarmID: farm.Identity(),
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(s.baseURL+"/nodes", "application/json", &buf)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	return "(unknown)", nil
}

type farmRegisterReq struct {
	ID   string `json:"farm_id"`
	Name string `json:"name"`
}

// RegisterFarm implements the IDStore interface
func (s *httpIDStore) RegisterFarm(farm modules.Identifier, name string, email string, wallet []string) (string, error) {
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(farmRegisterReq{
		ID:   farm.Identity(),
		Name: name,
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(s.baseURL+"/farms", "application/json", &buf)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	return name, nil
}
