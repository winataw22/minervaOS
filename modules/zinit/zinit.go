// Package zinit exposes function to interat with zinit service life cyle management
package zinit

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const defaultSocketPath = "/var/run/zinit.sock"

// Client is a client for zinit action
// it talks to zinit directly over its unis socket
type Client struct {
	conn net.Conn
	scan *bufio.Scanner
}

// New create a new Zinit client
func New(socket string) (*Client, error) {
	if socket == "" {
		socket = defaultSocketPath
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}

	scan := bufio.NewScanner(conn)
	return &Client{conn: conn, scan: scan}, nil
}

// Close closes the socket connection
func (c *Client) Close() error {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}

	c.conn = nil
	return nil
}

func (c *Client) cmd(cmd string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("not connected, call Connect() before executing command ")
	}
	if _, err := c.conn.Write([]byte(cmd)); err != nil {
		return "", err
	}
	if _, err := c.conn.Write([]byte("\n")); err != nil {
		return "", err
	}
	return c.readResponse()
}

func (c *Client) readResponse() (string, error) {
	var (
		count  uint64
		status string
		err    error
	)

	headers := map[string]string{}
	scanner := c.scan
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// end of headers section
			break
		}
		parts := strings.SplitN(line, ":", 2)
		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "error while reading socket")
	}

	count, err = strconv.ParseUint(headers["lines"], 10, 32)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse response length: %+v", headers)
	}

	status = headers["status"]

	var content strings.Builder
	for i := uint64(0); i < count; i++ {
		if !scanner.Scan() {
			break
		}

		if content.Len() > 0 {
			content.WriteByte('\n')
		}
		content.WriteString(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "error while reading socket")
	}

	if status == "error" {
		return "", fmt.Errorf(content.String())
	}

	return content.String(), nil
}
