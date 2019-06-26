package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// IDStore is the interface defining the
// client side of an identity store
type IDStore interface {
	RegisterNode(node Identifier, farm Identifier) error
	RegisterFarm(farm Identifier, name string) error
}

type httpIDStore struct {
	baseURL string
}

func NewHTTPIDStore(url string) IDStore {
	return &httpIDStore{baseURL: url}
}

type NodeRegisterReq struct {
	NodeID string `json:"node_id"`
	FarmID string `json:"farm_id"`
}

func (s *httpIDStore) RegisterNode(node Identifier, farm Identifier) error {
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(NodeRegisterReq{
		NodeID: node.Identity(),
		FarmID: farm.Identity(),
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(s.baseURL+"/nodes", "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	return nil
}

type FarmRegisterReq struct {
	ID   string `json:"farm_id"`
	Name string `json:"name"`
}

func (s *httpIDStore) RegisterFarm(farm Identifier, name string) error {
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(FarmRegisterReq{
		ID:   farm.Identity(),
		Name: name,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(s.baseURL+"/farms", "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	return nil
}
