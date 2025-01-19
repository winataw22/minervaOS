package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

// Client to a cloud hypervisor instance
type Client struct {
	client *retryablehttp.Client
}

type VMData struct {
	CPU     CPU
	Memory  MemMib
	PTYPath string
}

// NewClient creates a new instance of client
func NewClient(unix string) *Client {
	httpClient := retryablehttp.NewClient()
	httpClient.RetryMax = 5
	httpClient.HTTPClient.Transport = &http.Transport{
		Dial: func(network, _ string) (net.Conn, error) {
			return net.Dial("unix", unix)
		},
	}
	client := Client{
		client: httpClient,
	}

	return &client
}

// Shutdown shuts the machine down
func (c *Client) Shutdown(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://unix/api/v1/vm.shutdown", nil)
	if err != nil {
		return err
	}
	response, err := c.client.StandardClient().Do(request)
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
	response, err := c.client.StandardClient().Do(request)
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
	response, err := c.client.StandardClient().Do(request)
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
func (c *Client) Inspect(ctx context.Context) (VMData, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/api/v1/vm.info", nil)
	if err != nil {
		return VMData{}, err
	}
	request.Header.Add("content-type", "application/json")

	response, err := c.client.StandardClient().Do(request)
	if err != nil {
		return VMData{}, errors.Wrap(err, "error calling machine info")
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()

		return VMData{}, fmt.Errorf("got unexpected http code '%s' on machine info, Response: %s", response.Status, string(body))
	}

	var data struct {
		Config struct {
			CPU struct {
				Boot uint8 `json:"boot_vcpus"`
			} `json:"cpus"`
			Memory struct {
				Size int64 `json:"size"`
			} `json:"memory"`
			Serial struct {
				PTYPath string `json:"file"`
			} `json:"serial"`
		} `json:"config"`
	}

	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return VMData{}, errors.Wrap(err, "failed to parse machine information")
	}
	vmData := VMData{
		CPU:     CPU(data.Config.CPU.Boot),
		Memory:  MemMib(data.Config.Memory.Size / (1024 * 1024)),
		PTYPath: data.Config.Serial.PTYPath,
	}
	return vmData, nil
}
