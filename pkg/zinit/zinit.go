// Package zinit exposes function to interat with zinit service life cyle management
package zinit

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"

	"github.com/pkg/errors"
)

const defaultSocketPath = "/var/run/zinit.sock"

var unknownServiceRegex = regexp.MustCompile("^service name \".*\" unknown$")

// Client is a client for zinit action
// it talks to zinit directly over its unis socket
type Client struct {
	socket string
}

// New create a new Zinit client
func New(socket string) *Client {
	if socket == "" {
		socket = defaultSocketPath
	}
	return &Client{
		socket: socket,
	}
}

func Default() *Client {
	return New(defaultSocketPath)
}

func (c *Client) dial() (net.Conn, error) {
	return net.Dial("unix", c.socket)
}

func (c *Client) cmd(cmd string, out interface{}) error {
	conn, err := c.dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "%s\n", cmd); err != nil {
		return err
	}

	var result struct {
		State string          `json:"state"`
		Body  json.RawMessage `json:"body"`
	}

	if err := json.NewDecoder(conn).Decode(&result); err != nil {
		return err
	}

	if result.State == "error" {
		var msg string
		if err := json.Unmarshal(result.Body, &msg); err != nil {
			return errors.Wrapf(err, "failed to parse response error (%v) ", result.Body)
		}

		if unknownServiceRegex.Match([]byte(msg)) {
			return ErrUnknownService
		}
		return errors.New(msg)
	}

	if out == nil {
		return nil
	}

	return json.Unmarshal(result.Body, out)

}
