package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/pkg/errors"
)

// Client to a cloud hypervisor instance
type Client struct {
	client http.Client
}

// NewClient creates a new instance of client
func NewClient(unix string) *Client {
	client := Client{
		client: http.Client{
			Transport: &http.Transport{
				Dial: func(network, _ string) (net.Conn, error) {
					return net.Dial("unix", unix)
				},
			},
		},
	}

	return &client
}

// Shutdown shuts the machine down
func (c *Client) Shutdown(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://unix/api/v1/vm.shutdown", nil)
	if err != nil {
		return err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return errors.Wrap(err, "error calling machine shutdown")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("got unexpected http code '%s' on machine shutdown", response.Status)
	}

	return nil
}

func (c *Client) Pause(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://unix/api/v1/vm.pause", nil)
	if err != nil {
		return err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return errors.Wrap(err, "error calling machine pause")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("got unexpected http code '%s' on machine pause", response.Status)
	}

	return nil
}

func (c *Client) Resume(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://unix/api/v1/vm.resume", nil)
	if err != nil {
		return err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return errors.Wrap(err, "error calling machine pause")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("got unexpected http code '%s' on machine resume", response.Status)
	}

	return nil
}

// Inspect return information about the vm
func (c *Client) Inspect(ctx context.Context) (CPU, MemMib, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/api/v1/vm.info", nil)
	if err != nil {
		return 0, 0, err
	}
	request.Header.Add("content-type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return 0, 0, errors.Wrap(err, "error calling machine info")
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("got unexpected http code '%s' on machine info", response.Status)
	}

	var data struct {
		Config struct {
			CPUs struct {
				Boot uint8 `json:"boot_vcpus"`
			} `json:"cpus"`
			Memory struct {
				Size int64 `json:"size"`
			} `json:"memory"`
		} `json:"config"`
	}

	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return 0, 0, errors.Wrap(err, "failed to parse machine information")
	}

	return CPU(data.Config.CPUs.Boot),
		MemMib(data.Config.Memory.Size / (1024 * 1024)),
		nil
}
