package gateway

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/zinit"
	"gopkg.in/yaml.v2"
)

const (
	nncServicePrefix = "nnc-"
	nncStartPort     = 2000
)

// NNC holds (and extracts) information from nnc command
type NNC struct {
	ID   gridtypes.WorkloadID
	Exec string
}

func (n *NNC) arg(key string) (string, error) {
	parts, err := shlex.Split(n.Exec)
	if err != nil {
		return "", err
	}

	for i, part := range parts {
		if part == key {
			return parts[i+1], nil
		}
	}

	return "", fmt.Errorf("not found")
}

// port return port number this nnc instance is listening on
func (n *NNC) port() (uint16, error) {
	listen, err := n.arg("--listen")
	if err != nil {
		return 0, err
	}

	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		return 0, err
	}

	value, err := strconv.Atoi(port)
	if err != nil {
		return 0, err
	}

	return uint16(value), nil
}

// nncZinitPath return path to the nnc zinit config file given the name
func (g *gatewayModule) nncZinitPath(name string) string {
	return filepath.Join(g.volatile, zinitDir, fmt.Sprintf("%s.yaml", name))
}

// nncList lists all running instances of nnc. the map key is
// the instance configured listening port
func (g *gatewayModule) nncList() (map[uint16]NNC, error) {
	cl := zinit.Default()
	services, err := cl.List()
	if err != nil {
		return nil, err
	}
	nncs := make(map[uint16]NNC)
	// note: should we just instead list the config
	// from the known config directory?
	for name := range services {
		if !strings.HasPrefix(name, nncServicePrefix) {
			continue
		}
		id := strings.TrimPrefix(name, nncServicePrefix)

		cfg, err := cl.Get(name)
		if err != nil {
			return nil, err
		}

		nnc := NNC{
			ID:   gridtypes.WorkloadID(id),
			Exec: cfg.Exec,
		}

		port, err := nnc.port()
		if err != nil {
			return nil, err
		}

		nncs[port] = nnc
	}

	return nncs, nil
}

// nncGet return NNC instance by name
func (g *gatewayModule) nncGet(name string) (NNC, error) {
	cl := zinit.Default()
	cfg, err := cl.Get(name)
	if err != nil {
		return NNC{}, err
	}

	return NNC{
		ID:   gridtypes.WorkloadID(strings.TrimPrefix(name, nncServicePrefix)),
		Exec: cfg.Exec,
	}, nil

}

func (g *gatewayModule) nncName(id string) string {
	return fmt.Sprintf("%s%s", nncServicePrefix, id)
}

func (g *gatewayModule) nncFreePort() (uint16, error) {
	// TODO: this need to call while holding some lock
	// to avoid double allocation of the same port
	current, err := g.nncList()
	if err != nil {
		return 0, err
	}

	for {
		port := uint16(rand.Intn(math.MaxUint16-nncStartPort) + nncStartPort)
		if _, ok := current[port]; !ok {
			return port, nil
		}
	}
}

func (g *gatewayModule) nncCreateService(name string, service zinit.InitService) error {
	path := g.nncZinitPath(name)
	data, err := yaml.Marshal(service)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return errors.Wrap(err, "failed to create nnc service file")
	}

	// link under zinit /etc/zinit

	if err := os.Symlink(path, filepath.Join("/etc/zinit", fmt.Sprintf("%s.yaml", name))); err != nil {
		return errors.Wrap(err, "failed to create nnc service symlink")
	}

	return nil
}

// nncEnsure creates (or reuse) an nnc instance given the workload ID. the destination namespace and backend
// it return the backend that need to be configured in traefik.
func (g *gatewayModule) nncEnsure(wlID, namespace string, backend zos.Backend) (zos.Backend, error) {
	name := g.nncName(wlID)

	// reuse or find a new free IP
	var free uint16
	nnc, err := g.nncGet(name)
	if err == nil {
		// a service with the same name already exists
		// verify the backend?
		free, err = nnc.port()
		// we always destroy the service if
		// a one already exists with the same name
		// to allow updating the gw code.
		g.destroyNNC(wlID)
	} else if errors.Is(err, zinit.ErrUnknownService) {
		// if does not exist, just find a free port
		free, err = g.nncFreePort()
	} else if err != nil {
		return "", err
	}

	// this checks the error returned by the
	// free port allocation
	if err != nil {
		return "", err
	}

	target, err := backend.AsAddress()
	if err != nil {
		return "", err
	}

	be := zos.Backend(fmt.Sprintf("127.0.0.1:%d", free))

	cmd := []string{
		"ip", "netns", "exec", "public",
		"nnc",
		"--listen", string(be),
		"--namespace", filepath.Join("/var/run/netns/", namespace),
		"--target", target,
	}

	service := zinit.InitService{
		Exec: strings.Join(cmd, " "),
	}

	if err = g.nncCreateService(name, service); err != nil {
		return "", err
	}

	defer func() {
		if err != nil {
			g.destroyNNC(wlID)
		}
	}()

	if err = zinit.Default().Monitor(name); err != nil {
		return "", errors.Wrap(err, "failed to start nnc service")
	}

	return be, nil
}

// destroyNNC stops and clean up nnc instances
func (g *gatewayModule) destroyNNC(wlID string) {
	name := g.nncName(wlID)
	path := g.nncZinitPath(name)

	cl := zinit.Default()
	_ = cl.Destroy(10*time.Second, name)
	_ = os.Remove(path)
}
