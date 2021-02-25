package farmer

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/capacity"
)

// Location structure
type Location struct {
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Continent string  `json:"continent"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Node structure
type Node struct {
	ID       string            `json:"node_id"`
	HostName string            `json:"hostname"`
	FarmID   uint32            `json:"farm_id"`
	Secret   string            `json:"secret"`
	Location Location          `json:"location"`
	Capacity capacity.Capacity `json:"capacity"`
	// Type     string            `json:"type"`
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
