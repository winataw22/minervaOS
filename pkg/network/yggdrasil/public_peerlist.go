package yggdrasil

import (
	"encoding/json"
	"net/http"
)

//PeerListFallback is an hardcoded list of public yggdrasil node
// it is used to have some available peer to connect to when we failed to read the online public peer info
var PeerListFallback = Peers{
	{
		Endpoint: "tls://45.147.198.155:6010",
		Up:       true,
	},
	{
		Endpoint: "tcp://85.17.15.221:35239",
		Up:       true,
	},
	{
		Endpoint: "tcp://51.255.223.60:64982",
		Up:       true,
	},
}

// NodeInfo is the know information about an yggdrasil public node
type NodeInfo struct {
	Endpoint   string `json:"-"`
	Up         bool   `json:"up"`
	BoxPubKey  string `json:"key"`
	LastSeen   int    `json:"last_seen"`
	ProtoMinor int    `json:"proto_minor"`
}

// Peers is a peers list
type Peers []NodeInfo

// FetchPeerList download the list of public yggdrasil peer from https://publicpeers.neilalexander.dev/publicnodes.json
func FetchPeerList() (Peers, error) {
	//pl := PeerList{}
	pl := map[string]map[string]NodeInfo{}

	resp, err := http.Get("https://publicpeers.neilalexander.dev/publicnodes.json")
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&pl); err != nil {
		return nil, err
	}

	var peers Peers
	for _, nodes := range pl {
		for endpoint, node := range nodes {
			if node.ProtoMinor != 4 {
				continue
			}

			node.Endpoint = endpoint
			peers = append(peers, node)
		}
	}

	return peers, nil
}

// Ups return all the peers that are marked up from the PeerList p
func (p Peers) Ups() Peers {
	var peers Peers
	for _, n := range p {
		if n.Up {
			peers = append(peers, n)
		}
	}

	return peers
}
