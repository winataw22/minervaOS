package farmer

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Location structure
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Node structure
type Node struct {
	ID       string             `json:"node"`
	FarmID   uint32             `json:"farm"`
	Secret   string             `json:"secret"`
	Location Location           `json:"location"`
	Capacity gridtypes.Capacity `json:"capacity"`
}

// NodeRegister register node
func (c *Client) NodeRegister(node Node) error {
	url := c.path("nodes")
	body, err := c.serialize(node)
	if err != nil {
		return errors.Wrap(err, "failed to create request body")
	}

	response, err := http.Post(url, contentType, body)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	return c.response(response, nil, http.StatusCreated, http.StatusNotModified)
}

// GatewayRegister registers a node as a gateway
func (c *Client) GatewayRegister(node Node) error {
	url := c.path("gateways")
	body, err := c.serialize(node)
	if err != nil {
		return errors.Wrap(err, "failed to create request body")
	}

	response, err := http.Post(url, contentType, body)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	return c.response(response, nil, http.StatusCreated, http.StatusNotModified)
}
