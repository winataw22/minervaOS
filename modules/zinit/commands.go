package zinit

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ServiceState is a type representing the state of a service managed by zinit
type ServiceState int

const (
	// ServiceStatusUnknown is return when we cannot determine the status of a service
	ServiceStatusUnknown ServiceState = iota
	// ServiceStatusRunning is return when we a service process is running and healthy
	ServiceStatusRunning
	// ServiceStatusHalted is return when we a service is not running because stopped explicitly
	ServiceStatusHalted
	// ServiceStatusSuccess is return when a one shot service exited without errors
	ServiceStatusSuccess
	// ServiceStatusError is return when we a service is not running while it should
	// this means something has made the service crash
	ServiceStatusError
)

var strToServiceState = map[string]ServiceState{
	"unknown": ServiceStatusUnknown,
	"running": ServiceStatusRunning,
	"success": ServiceStatusSuccess,
	"halted":  ServiceStatusHalted,
	"error":   ServiceStatusError,
}

type ServiceTarget int

const (
	ServiceTargetUp ServiceTarget = iota
	ServiceTargetDown
)

var strToServiceTarget = map[string]ServiceTarget{
	"up":   ServiceTargetUp,
	"down": ServiceTargetDown,
}

type ServiceStatus struct {
	Name   string
	Pid    int
	State  ServiceState
	Target ServiceTarget
}

// List returns all the service monitored and their status
func (c *ZinitClient) List() (map[string]ServiceState, error) {
	resp, err := c.cmd("list")
	if err != nil {
		return nil, err
	}

	return parseList(resp)
}

func parseList(s string) (map[string]ServiceState, error) {
	lines := strings.Split(s, "\n")
	l := make(map[string]ServiceState, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		service := strings.TrimSpace(parts[0])
		status := strings.TrimSpace(parts[1])
		s, ok := strToServiceState[strings.ToLower(status)]
		if !ok {
			return nil, fmt.Errorf("unsupported service status: %v", status)
		}
		l[service] = s
	}

	return l, nil
}

// Status returns the status of a service
func (c *ZinitClient) Status(service string) (ServiceStatus, error) {
	resp, err := c.cmd(fmt.Sprintf("status %s", service))
	if err != nil {
		return ServiceStatus{}, err
	}

	return parseStatus(resp)
}

func parseStatus(s string) (ServiceStatus, error) {
	status := ServiceStatus{}
	lines := strings.Split(s, "\n")

	entries := map[string]string{}
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		entries[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	var (
		ok  bool
		err error
	)

	status.Name, ok = entries["name"]
	if !ok {
		return status, fmt.Errorf("name field not found in response")
	}

	spid, ok := entries["pid"]
	if !ok {
		return status, fmt.Errorf("pid field not found in response")
	}
	status.Pid, err = strconv.Atoi(spid)
	if err != nil {
		return status, err
	}

	state, ok := entries["state"]
	if !ok {
		return status, fmt.Errorf("state field not found in response")
	}
	status.State, ok = strToServiceState[strings.ToLower(state)]
	if !ok {
		return status, fmt.Errorf("state value not supported: %v", state)
	}

	target, ok := entries["target"]
	if !ok {
		return status, fmt.Errorf("target field not found in response")
	}
	status.Target, ok = strToServiceTarget[strings.ToLower(target)]
	if !ok {
		return status, fmt.Errorf("target value not supported: %v", target)
	}

	return status, nil
}

// Start start service. has no effect if the service is already running
func (c *ZinitClient) Start(service string) error {
	_, err := c.cmd(fmt.Sprintf("start %s", service))
	return err
}

// Stop stops a service
func (c *ZinitClient) Stop(service string) error {
	_, err := c.cmd(fmt.Sprintf("stop %s", service))
	return err
}

// Monitor starts monitoring a service
func (c *ZinitClient) Monitor(service string) error {
	_, err := c.cmd(fmt.Sprintf("monitor %s", service))
	return err
}

// Fortget forget a service. you can only forget a stopped service
func (c *ZinitClient) forget(service string) error {
	_, err := c.cmd(fmt.Sprintf("forget %s", service))
	return err
}

// Kill sends a signal to a running service.
func (c *ZinitClient) Kill(service string, sig os.Signal) error {
	_, err := c.cmd(fmt.Sprintf("kill %s %s", service, sig.String()))
	return err
}
