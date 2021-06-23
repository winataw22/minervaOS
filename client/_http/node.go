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
	url := "http://[" + n.ip.String() + "]:2021/api/v1/" + filepath.Join(path...)
	return url
}

// Deploy sends the workload for node to deploy. On success means the node
// accepted the workload (it passed validation), doesn't mean it has been
// deployed. the user then can pull on the workload status until it passes (or fail)
func (n *NodeClient) Deploy(dl *gridtypes.Deployment, update bool) error {
	dl.TwinID = n.client.id
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(dl); err != nil {
		return errors.Wrap(err, "failed to serialize workload")
	}

	url := n.url("deployment")
	m := http.MethodPost
	if update {
		m = http.MethodPut
	}

	request, err := http.NewRequest(m, url, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	if err := n.client.authorize(request); err != nil {
		return errors.Wrap(err, "failed to sign request")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	if err := n.response(response, nil, http.StatusAccepted); err != nil {
		return err
	}

	return nil
}

// Get get a workload by id
func (n *NodeClient) Get(twin, deployment uint32) (dl gridtypes.Deployment, err error) {
	url := n.url("deployment", fmt.Sprint(twin), fmt.Sprint(deployment))

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return dl, errors.Wrap(err, "failed to build request")
	}

	if err := n.client.authorize(request); err != nil {
		return dl, errors.Wrap(err, "failed to sign request")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return dl, err
	}

	if err := n.response(response, &dl, http.StatusOK); err != nil {
		return dl, err
	}

	return dl, nil
}

// Delete deletes a workload by id
func (n *NodeClient) Delete(twin, deployment uint32) (err error) {
	url := n.url("deployment", fmt.Sprint(twin), fmt.Sprint(deployment))

	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	if err := n.client.authorize(request); err != nil {
		return errors.Wrap(err, "failed to sign request")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	if err := n.response(response, nil, http.StatusAccepted); err != nil {
		return err
	}

	return nil
}

func (n *NodeClient) Counters() (total gridtypes.Capacity, used gridtypes.Capacity, err error) {
	url := n.url("counters")

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return total, used, errors.Wrap(err, "failed to build request")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return total, used, err
	}
	var result struct {
		Total gridtypes.Capacity `json:"total"`
		Used  gridtypes.Capacity `json:"used"`
	}
	if err := n.response(response, &result, http.StatusOK); err != nil {
		return total, used, err
	}

	return result.Total, result.Used, nil
}
