package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"

	"github.com/threefoldtech/zos/pkg/geoip"

	"github.com/threefoldtech/zos/pkg"
)

// IDStore is the interface defining the
// client side of an identity store
type IDStore interface {
	RegisterNode(node pkg.Identifier, farm pkg.Identifier, version string, loc geoip.Location) (string, error)
	RegisterFarm(farm pkg.Identifier, name string, email string, wallet []string) (string, error)
}

type httpIDStore struct {
	baseURL string
}

// NewHTTPIDStore returns a HTTP IDStore client
func NewHTTPIDStore(url string) IDStore {
	return &httpIDStore{baseURL: url}
}

// RegisterNode implements the IDStore interface
func (s *httpIDStore) RegisterNode(node pkg.Identifier, farm pkg.Identifier, version string, loc geoip.Location) (string, error) {
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(directory.TfgridNode2{
		NodeID:    node.Identity(),
		FarmID:    farm.Identity(),
		OsVersion: version,
		Location: directory.TfgridLocation1{
			City:      loc.City,
			Country:   loc.Country,
			Continent: loc.Continent,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitute,
		},
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
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RegisterFarm implements the IDStore interface
func (s *httpIDStore) RegisterFarm(farm pkg.Identifier, name string, email string, wallet []string) (string, error) {
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
