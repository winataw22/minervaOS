package farmer

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Consumption struct
type Consumption struct {
	Workloads map[gridtypes.WorkloadType][]gridtypes.WorkloadID `json:"workloads"`
	Capacity  gridtypes.Capacity                                `json:"capacity"`
}

// Report is a user report
type Report struct {
	Timestamp   int64                  `json:"timestamp"`
	Signature   string                 `json:"signature"`
	Consumption map[uint32]Consumption `json:"consumption"`
}

// Challenge build a report challenge for signing
func (r *Report) Challenge(w io.Writer) error {
	var total gridtypes.Capacity
	for _, consumption := range r.Consumption {
		total.Add(&consumption.Capacity)
	}

	if _, err := fmt.Fprintf(w, "%d", r.Timestamp); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", total.CRU); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", total.MRU); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", total.SRU); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", total.HRU); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", total.IPV4U); err != nil {
		return err
	}

	return nil
}

// Sign the report given sk
func (r *Report) Sign(sk ed25519.PrivateKey) error {
	var buf bytes.Buffer
	if err := r.Challenge(&buf); err != nil {
		return errors.Wrap(err, "failed to build signing challenge")
	}

	signature := ed25519.Sign(sk, buf.Bytes())
	r.Signature = hex.EncodeToString(signature)
	return nil
}

// NodeReport send usage reports to farmer
func (c *Client) NodeReport(nodeID string, report Report) error {
	url := c.path("nodes", nodeID, "reports")

	body, err := c.serialize(report)
	if err != nil {
		return errors.Wrap(err, "failed to create request body")
	}

	response, err := http.Post(url, contentType, body)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	return c.response(response, nil, http.StatusCreated, http.StatusOK)
}
