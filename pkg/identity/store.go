package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"

	"github.com/threefoldtech/zos/pkg/geoip"

	"github.com/threefoldtech/zos/pkg"
)

// IDStore is the interface defining the
// client side of an identity store
type IDStore interface {
	RegisterNode(node pkg.Identifier, farm pkg.FarmID, version string, loc geoip.Location) (string, error)
	RegisterFarm(tid uint64, name string, email string, wallet []string) (pkg.FarmID, error)
}

type httpIDStore struct {
	baseURL string
}

// NewHTTPIDStore returns a HTTP IDStore client
func NewHTTPIDStore(url string) IDStore {
	return &httpIDStore{baseURL: url}
}

// RegisterNode implements the IDStore interface
func (s *httpIDStore) RegisterNode(node pkg.Identifier, farm pkg.FarmID, version string, loc geoip.Location) (string, error) {
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(directory.TfgridNode2{
		NodeID:    node.Identity(),
		FarmID:    uint64(farm),
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

// RegisterFarm implements the IDStore interface
func (s *httpIDStore) RegisterFarm(tid uint64, name string, email string, wallet []string) (pkg.FarmID, error) {
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(directory.TfgridFarm1{
		ThreebotID:      tid,
		Name:            name,
		Email:           email,
		WalletAddresses: wallet,
	})
	if err != nil {
		return 0, err
	}

	resp, err := http.Post(s.baseURL+"/farms", "application/json", &buf)
	if err != nil {
		return 0, err
	}
	defer func() {
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		msg, _ := ioutil.ReadAll(resp.Body)
		return 0, fmt.Errorf("wrong response status code received: (%s) %s", string(msg), resp.Status)
	}

	id := struct {
		ID pkg.FarmID `json:"id"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&id); err != nil {
		return 0, err
	}

	return id.ID, nil
}
