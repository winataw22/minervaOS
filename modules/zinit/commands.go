package zinit

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// ServiceState represents the state of a service managed by zinit
type ServiceState string

const (
	// ServiceStatusUnknown is return when we cannot determine the status of a service
	ServiceStatusUnknown ServiceState = "unknown"
	// ServiceStatusRunning is return when we a service process is running and healthy
	ServiceStatusRunning = "running"
	// ServiceStatusHalted is return when we a service is not running because stopped explicitly
	ServiceStatusHalted = "halted"
	// ServiceStatusSuccess is return when a one shot service exited without errors
	ServiceStatusSuccess = "success"
	// ServiceStatusError is return when we a service is not running while it should
	// this means something has made the service crash
	ServiceStatusError = "error"
)

// UnmarshalYAML implements the  yaml.Unmarshaler interface
func (s *ServiceState) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	if err := unmarshal(&buf); err != nil {
		return err
	}
	*s = ServiceState(strings.ToLower(buf))
	return nil
}

// ServiceTarget represents the desired state of a service
type ServiceTarget string

// UnmarshalYAML implements the  yaml.Unmarshaler interface
func (s *ServiceTarget) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	if err := unmarshal(&buf); err != nil {
		return err
	}
	*s = ServiceTarget(strings.ToLower(buf))
	return nil
}

const (
	// ServiceTargetUp means the service has been asked to start
	ServiceTargetUp ServiceTarget = "up"
	// ServiceTargetDown means the service has been asked to stop
	ServiceTargetDown = "down"
)

// ServiceStatus represent the status of a service
type ServiceStatus struct {
	Name   string
	Pid    int
	State  ServiceState
	Target ServiceTarget
	// Deps is the list of dependent services
	// all dependencies needs to be running before a service
	// can starts
	Deps map[string]string
}

// List returns all the service monitored and their status
func (c *Client) List() (map[string]ServiceState, error) {
	resp, err := c.cmd("list")
	if err != nil {
		return nil, err
	}

	return parseList(resp)
}

func parseList(s string) (map[string]ServiceState, error) {
	l := make(map[string]ServiceState)
	if err := yaml.Unmarshal([]byte(s), &l); err != nil {
		return nil, err
	}
	return l, nil
}

// Status returns the status of a service
func (c *Client) Status(service string) (ServiceStatus, error) {
	resp, err := c.cmd(fmt.Sprintf("status %s", service))
	if err != nil {
		return ServiceStatus{}, err
	}

	return parseStatus(resp)
}

func parseStatus(s string) (ServiceStatus, error) {
	status := ServiceStatus{}
	if err := yaml.Unmarshal([]byte(s), &status); err != nil {
		return status, err
	}
	return status, nil
}

// Start start service. has no effect if the service is already running
func (c *Client) Start(service string) error {
	_, err := c.cmd(fmt.Sprintf("start %s", service))
	return err
}

// Stop stops a service
func (c *Client) Stop(service string) error {
	_, err := c.cmd(fmt.Sprintf("stop %s", service))
	return err
}

// Monitor starts monitoring a service
func (c *Client) Monitor(service string) error {
	_, err := c.cmd(fmt.Sprintf("monitor %s", service))
	return err
}

// Fortget forget a service. you can only forget a stopped service
func (c *Client) forget(service string) error {
	_, err := c.cmd(fmt.Sprintf("forget %s", service))
	return err
}

// Kill sends a signal to a running service.
func (c *Client) Kill(service string, sig os.Signal) error {
	_, err := c.cmd(fmt.Sprintf("kill %s %s", service, sig.String()))
	return err
}
