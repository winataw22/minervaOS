package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	successCodes = []int{http.StatusOK}
)

// NodeClient is a client of the node
type NodeClient struct {
	client *Client
	ip     net.IP
}

func (n *NodeClient) response(r *http.Response, o interface{}, codes ...int) error {
	defer r.Body.Close()
	if len(codes) == 0 {
		codes = successCodes
	}

	in := func(i int, l []int) bool {
		for _, v := range l {
			if v == i {
				return true
			}
		}
		return false
	}

	if !in(r.StatusCode, codes) {
		msg, _ := ioutil.ReadAll(r.Body)
		return fmt.Errorf("invalid response (%s): %s", r.Status, string(msg))
	}

	defer func() {
		ioutil.ReadAll(r.Body)
	}()

	if o != nil {
		return json.NewDecoder(r.Body).Decode(o)
	}

	return nil
}

func (n *NodeClient) url(path ...string) string {
	url := "http://[%s]:2021/api/v1/" + filepath.Join(path...) + "/"
	return url
}

// Deploy sends the workload for node to deploy. On success means the node
// accepted the workload (it passed validation), doesn't mean it has been
// deployed. the user then can pull on the workload status until it passes (or fail)
func (n *NodeClient) Deploy(wl *gridtypes.Workload) (wid string, err error) {
	wl.ID = n.client.id
	if err := wl.Sign(n.client.sk); err != nil {
		return wid, errors.Wrap(err, "failed to sign the workload")
	}

	if err := wl.Sign(n.client.sk); err != nil {
		return wid, errors.Wrap(err, "failed to sign signature")
	}

	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(wl); err != nil {
		return wid, errors.Wrap(err, "failed to serialize workload")
	}

	url := n.url("workloads")

	request, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return wid, errors.Wrap(err, "failed to build request")
	}

	if err := n.client.signer.Sign(request); err != nil {
		return wid, errors.Wrap(err, "failed to sign request")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	if err := n.response(response, &wid, http.StatusAccepted); err != nil {
		return wid, err
	}

	return wid, nil
}
