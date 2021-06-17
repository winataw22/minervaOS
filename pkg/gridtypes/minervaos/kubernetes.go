package zos

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var vmSize = map[uint8]gridtypes.Capacity{
	1: {
		CRU: 1,
		MRU: 2 * gridtypes.Gigabyte,
	},
	2: {
		CRU: 2,
		MRU: 4 * gridtypes.Gigabyte,
	},
	3: {
		CRU: 2,
		MRU: 8 * gridtypes.Gigabyte,
	},
	4: {
		CRU: 2,
		MRU: 5 * gridtypes.Gigabyte,
	},
	5: {
		CRU: 2,
		MRU: 8 * gridtypes.Gigabyte,
	},
	6: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
	},
	7: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
	},
	8: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
	},
	9: {
		CRU: 8,
		MRU: 32 * gridtypes.Gigabyte,
	},
	10: {
		CRU: 8,
		MRU: 32 * gridtypes.Gigabyte,
	},
	11: {
		CRU: 8,
		SRU: 800 * gridtypes.Gigabyte,
	},
	12: {
		CRU: 1,
		MRU: 64 * gridtypes.Gigabyte,
	},
	13: {
		CRU: 1,
		MRU: 64 * gridtypes.Gigabyte,
	},
	14: {
		CRU: 1,
		SRU: 800 * gridtypes.Gigabyte,
	},
	15: {
		CRU: 1,
		SRU: 25 * gridtypes.Gigabyte,
	},
	16: {
		CRU: 2,
		SRU: 50 * gridtypes.Gigabyte,
	},
	17: {
		CRU: 4,
		SRU: 50 * gridtypes.Gigabyte,
	},
	18: {
		CRU: 1,
		MRU: 1 * gridtypes.Gigabyte,
	},
}

// Kubernetes reservation data
type Kubernetes struct {
	ZMachine `json:",inline"`
	// ClusterSecret is the hex encoded encrypted(?) cluster secret.
	ClusterSecret string `json:"cluster_secret"`
	// MasterIPs define the URL's for the kubernetes master nodes. If this
	// list is empty, this node is considered to be a master node.
	MasterIPs []net.IP `json:"master_ips"`

	DatastoreEndpoint     string `json:"datastore_endpoint"`
	DisableDefaultIngress bool   `json:"disable_default_ingress"`
}

// Challenge implementation
func (k Kubernetes) Challenge(b io.Writer) error {
	k.ZMachine.Challenge(b)

	if _, err := fmt.Fprintf(b, "%s", k.ClusterSecret); err != nil {
		return err
	}
	for _, ip := range k.MasterIPs {
		if _, err := fmt.Fprintf(b, "%s", ip.String()); err != nil {
			return err
		}
	}

	return nil
}

// Valid implementation
func (k Kubernetes) Valid(getter gridtypes.WorkloadGetter) error {
	err := k.ZMachine.Valid(getter)
	if err != nil {
		return err
	}
	if strings.ContainsAny(k.ClusterSecret, " \t\r\n\f") {
		return errors.New("cluster secret shouldn't contain whitespace chars")
	}

	for _, ip := range k.MasterIPs {
		if ip.To4() == nil && ip.To16() == nil {
			return errors.New("invalid master IP")
		}
	}
	return nil
}

// KubernetesResult result returned by k3s reservation
type KubernetesResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}
