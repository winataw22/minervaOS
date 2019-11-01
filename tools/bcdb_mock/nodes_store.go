package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/schema"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
)

type nodeStore struct {
	Nodes []*directory.TfgridNode2 `json:"nodes"`
	m     sync.RWMutex
}

func loadNodeStore() (*nodeStore, error) {
	store := &nodeStore{
		Nodes: []*directory.TfgridNode2{},
	}
	f, err := os.OpenFile("nodes.json", os.O_RDONLY, 0660)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		return store, err
	}
	return store, nil
}

func (s *nodeStore) Save() error {
	s.m.RLock()
	defer s.m.RUnlock()

	f, err := os.OpenFile("nodes.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(s); err != nil {
		return err
	}
	return nil
}

func (s *nodeStore) List() []*directory.TfgridNode2 {
	s.m.RLock()
	defer s.m.RUnlock()
	out := make([]*directory.TfgridNode2, len(s.Nodes))

	copy(out, s.Nodes)
	return out
}

func (s *nodeStore) Get(nodeID string) (*directory.TfgridNode2, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	for _, n := range s.Nodes {
		if n.NodeID == nodeID {
			return n, nil
		}
	}
	return nil, fmt.Errorf("node %s not found", nodeID)
}

func (s *nodeStore) Add(node directory.TfgridNode2) error {
	s.m.Lock()
	defer s.m.Unlock()

	for i, n := range s.Nodes {
		if n.NodeID == node.NodeID {
			s.Nodes[i].FarmID = node.FarmID
			s.Nodes[i].OsVersion = node.OsVersion
			s.Nodes[i].Location = node.Location
			s.Nodes[i].Updated = schema.Date{Time: time.Now()}
			return nil
		}
	}

	node.Created = schema.Date{Time: time.Now()}
	node.Updated = schema.Date{Time: time.Now()}
	s.Nodes = append(s.Nodes, &node)
	return nil
}

func (s *nodeStore) updateTotalCapacity(nodeID string, cap directory.TfgridNodeResourceAmount1) error {
	return s.updateCapacity(nodeID, "total", cap)
}
func (s *nodeStore) updateReservedCapacity(nodeID string, cap directory.TfgridNodeResourceAmount1) error {
	return s.updateCapacity(nodeID, "reserved", cap)
}
func (s *nodeStore) updateUsedCapacity(nodeID string, cap directory.TfgridNodeResourceAmount1) error {
	return s.updateCapacity(nodeID, "used", cap)
}

func (s *nodeStore) updateCapacity(nodeID string, t string, cap directory.TfgridNodeResourceAmount1) error {
	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	switch t {
	case "total":
		node.TotalResources = cap
	case "reserved":
		node.ReservedResources = cap
	case "used":
		node.UsedResources = cap
	default:
		return fmt.Errorf("unsupported capacity type: %v", t)
	}

	return nil
}

func (s *nodeStore) updateUptime(nodeID string, uptime int64) error {
	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	node.Uptime = uptime
	node.Updated = schema.Date{Time: time.Now()}

	return nil
}

func (s *nodeStore) StoreProof(nodeID string, dmi dmi.DMI, disks capacity.Disks) error {
	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	proof := directory.TfgridNodeProof1{
		Created: schema.Date{Time: time.Now()},
	}

	proof.Hardware = map[string]interface{}{
		"sections": dmi.Sections,
		"tooling":  dmi.Tooling,
	}
	proof.HardwareHash, err = hashProof(proof.Hardware)
	if err != nil {
		return err
	}

	proof.Disks = map[string]interface{}{
		"aggregator":  disks.Aggregator,
		"environment": disks.Environment,
		"devices":     disks.Devices,
		"tool":        disks.Tool,
	}
	proof.DiskHash, err = hashProof(proof.Disks)
	if err != nil {
		return err
	}

	// don't save the proof if we already have one with the same
	// hash/content
	for _, p := range node.Proofs {
		if proof.Equal(p) {
			return nil
		}
	}

	node.Proofs = append(node.Proofs, proof)
	return nil
}

func (s *nodeStore) SetInterfaces(nodeID string, ifaces []directory.TfgridNodeIface1) error {
	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	node.Ifaces = ifaces
	return nil
}

func (s *nodeStore) SetPublicConfig(nodeID string, cfg directory.TfgridNodePublicIface1) error {
	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	node.PublicConfig = &cfg
	return nil
}

func (s *nodeStore) SetWGPorts(nodeID string, ports []uint) error {
	node, err := s.Get(nodeID)
	if err != nil {
		return err
	}

	node.WGPorts = ports
	return nil
}

func (s *nodeStore) Requires(key string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeID, ok := mux.Vars(r)[key]
		if !ok {
			// programming error, we should panic in this case
			panic("invalid node-id key")
		}

		_, err := s.Get(nodeID)
		if err != nil {
			// node not found
			httpError(w, errors.Wrapf(err, "node not found: %s", nodeID), http.StatusNotFound)
			return
		}

		handler(w, r)
	}
}

// hashProof return the hex encoded md5 hash of the json encoded version of p
func hashProof(p map[string]interface{}) (string, error) {

	// we are trying to have always produce same hash for same content of p
	// so we convert the map into a list so we can sort
	// the key and workaround the fact that maps are not sorted

	type kv struct {
		k string
		v interface{}
	}

	kvs := make([]kv, len(p))
	for k, v := range p {
		kvs = append(kvs, kv{k: k, v: v})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].k < kvs[j].k })

	b, err := json.Marshal(kvs)
	if err != nil {
		return "", err
	}
	h := md5.New()
	bh := h.Sum(b)
	return fmt.Sprintf("%x", bh), nil
}
