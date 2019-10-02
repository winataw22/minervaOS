package zinit

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	statusRegex = regexp.MustCompile(`^(\w+)(?:\((.+)\))?$`)
)

// PossibleState represents the state of a service managed by zinit
type PossibleState string

const (
	// ServiceStateUnknown is return when we cannot determine the status of a service
	ServiceStateUnknown PossibleState = "unknown"
	// ServiceStateRunning is return when we a service process is running and healthy
	ServiceStateRunning = "running"
	// ServiceStateBlocked  is returned if the service can't start because of an unsatisfied dependency
	ServiceStateBlocked = "blocked"
	// ServiceStateSpawned service has started, but zinit is not sure about its status yet.
	// this is usually a short-lived status, unless a test command is provided. In that case
	// the spawned state will only go to success if the test pass
	ServiceStateSpawned = "spawned"
	// ServiceStateSuccess is return when a one shot service exited without errors
	ServiceStateSuccess = "success"
	// ServiceStateError is return when we a service exit with an error (exit code != 0)
	ServiceStateError = "error"
	//ServiceStateFailure is set of zinit can not spawn a service in the first place
	//due to a missing executable for example. Unlike `error` which is returned if the
	//service itself exits with an error.
	ServiceStateFailure = "failure"
)

// ServiceState describes the service state
type ServiceState struct {
	state  PossibleState
	reason string
}

// UnmarshalYAML implements the  yaml.Unmarshaler interface
func (s *ServiceState) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	if err := unmarshal(&buf); err != nil {
		return err
	}

	m := statusRegex.FindStringSubmatch(buf)
	if len(m) != 3 {
		return fmt.Errorf("invalid service state value")
	}

	s.state = PossibleState(strings.ToLower(m[1]))
	s.reason = strings.ToLower(m[2])
	return nil
}

func (s *ServiceState) String() string {
	var buf strings.Builder
	if len(s.state) != 0 {
		buf.WriteString(string(s.state))
	} else {
		buf.WriteString(string(ServiceStateUnknown))
	}

	if len(s.reason) != 0 {
		buf.WriteByte('(')
		buf.WriteString(s.reason)
		buf.WriteByte(')')
	}

	return buf.String()
}

// Is checks if service state is equal to a possible state
func (s *ServiceState) Is(state PossibleState) bool {
	return strings.EqualFold(string(s.state), string(state))
}

// Exited is true if the service state in a (stopped) state
func (s *ServiceState) Exited() bool {
	return s.Is(ServiceStateSuccess) || s.Is(ServiceStateError) || s.Is(ServiceStateFailure)
}

// MarshalYAML implements the  yaml.Unmarshaler interface
func (s *ServiceState) MarshalYAML() (interface{}, error) {
	return s.String(), nil
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

// StartWait starts a service and wait until its running, or until the timeout
// (seconds) pass. If timedout, the method returns an error if the service is not running
// timout of 0 means no wait. (similar to Stop)
// timout is a min of 1 second
func (c *Client) StartWait(timeout time.Duration, service string) error {
	if err := c.Start(service); err != nil {
		return err
	}

	if timeout == 0 {
		return nil
	}

	deadline := time.After(timeout)

	for {
		select {
		case <-deadline:
			status, err := c.Status(service)
			if err != nil {
				return err
			}

			if status.State.Exited() {
				return fmt.Errorf("service '%s' did not start in time", service)
			}

			return nil
		default:
			status, err := c.Status(service)
			if err != nil {
				return err
			}

			if status.Target != ServiceTargetUp {
				// it means some other entity (another client or command line)
				// has set the service back to down. I think we should immediately return
				// with an error instead.
				return fmt.Errorf("expected service target should be UP. found DOWN")
			}

			if status.State.Is(ServiceStateRunning) || status.State.Is(ServiceStateSuccess) {
				return nil
			}

			<-time.After(1 * time.Second)
		}
	}
}

// StopWait stops a service and wait until it exits, or until the timeout
// (seconds) pass. If timedout, the service is killed with -9.
// timout of 0 means no wait. (similar to Stop)
// timout is a min of 1 second
func (c *Client) StopWait(timeout time.Duration, service string) error {
	if err := c.Stop(service); err != nil {
		return err
	}

	if timeout == 0 {
		return nil
	}

	deadline := time.After(timeout)

	for {
		select {
		case <-deadline:
			return c.Kill(service, syscall.SIGKILL)
		default:
			status, err := c.Status(service)
			if err != nil {
				return err
			}

			if status.Target != ServiceTargetDown {
				// it means some other entity (another client or command line)
				// has set the service back to up. I think we should immediately return
				// with an error instead.
				return fmt.Errorf("expected service target should be DOWN. found UP")
			}

			if status.State.Exited() {
				return nil
			}
			<-time.After(1 * time.Second)
		}
	}
}

// Monitor starts monitoring a service
func (c *Client) Monitor(service string) error {
	_, err := c.cmd(fmt.Sprintf("monitor %s", service))
	if err != nil && strings.Contains(err.Error(), "already monitored") {
		return nil
	}
	return err
}

// Forget forgets a service. you can only forget a stopped service
func (c *Client) Forget(service string) error {
	_, err := c.cmd(fmt.Sprintf("forget %s", service))
	return err
}

// Kill sends a signal to a running service.
func (c *Client) Kill(service string, sig os.Signal) error {
	_, err := c.cmd(fmt.Sprintf("kill %s %s", service, sig.String()))
	return err
}
