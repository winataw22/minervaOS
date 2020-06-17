package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
)

// YggdrasilAdminAPI is a client that talk to the yggdrasil admin API
type YggdrasilAdminAPI struct {
	connectionURI string
	conn          net.Conn
}

type response struct {
	Status   string                 `json:"status,omitempty"`
	Request  map[string]interface{} `json:"request,omitempty"`
	Response json.RawMessage        `json:"response,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// NewYggdrasil create a new yggdrasil admin API client by connecting to connectionURI
func NewYggdrasil(connectionURI string) *YggdrasilAdminAPI {
	return &YggdrasilAdminAPI{
		connectionURI: connectionURI,
	}
}

// Connect opens the connection to the API
func (y *YggdrasilAdminAPI) Connect() error {
	u, err := url.Parse(y.connectionURI)
	if err != nil {
		return fmt.Errorf("wrong format for connection URI %v: %w", y.connectionURI, err)
	}

	conn, err := net.Dial(strings.ToLower(u.Scheme), u.Host)
	if err != nil {
		return err
	}

	y.conn = conn
	return nil
}

// Close closes all connection to the API
func (y *YggdrasilAdminAPI) Close() error {
	if y.conn != nil {
		return y.conn.Close()
	}
	return nil
}

func (y *YggdrasilAdminAPI) execReq(req string) (response, error) {
	resp := response{}
	if _, err := y.conn.Write([]byte(req)); err != nil {
		return resp, err
	}

	if err := json.NewDecoder(y.conn).Decode(&resp); err != nil {
		return resp, err
	}

	if resp.Error != "" {
		log.Printf("%s", string(resp.Response))
		return resp, fmt.Errorf(resp.Error)
	}

	return resp, nil
}

// GetSelf implement the getself Admin API call
func (y *YggdrasilAdminAPI) GetSelf() (nodeinfo NodeInfo, err error) {

	resp, err := y.execReq(`{"keepalive":true, "request":"getSelf"}`)
	if err != nil {
		return
	}

	data := struct {
		Self map[string]NodeInfo `json:"self"`
	}{}
	if err = json.Unmarshal(resp.Response, &data); err != nil {
		return
	}

	for k := range data.Self {
		nodeinfo = data.Self[k]
		nodeinfo.IPv6Addr = k
		break
	}

	return nodeinfo, nil
}

// GetPeers implement the getpeers Admin API call
func (y *YggdrasilAdminAPI) GetPeers() ([]Peer, error) {
	resp, err := y.execReq(`{"keepalive":true, "request":"getpeers"}`)
	if err != nil {
		return nil, err
	}

	data := struct {
		Peers map[string]Peer `json:"peers"`
	}{}
	if err = json.Unmarshal(resp.Response, &data); err != nil {
		return nil, err
	}

	peers := make([]Peer, 0, len(data.Peers))

	for k := range data.Peers {
		peer := data.Peers[k]
		peer.IPv6Addr = k
		peers = append(peers, peer)
	}

	return peers, nil
}

// AddPeer implement the addpeer Admin API call
func (y *YggdrasilAdminAPI) AddPeer(uri string) ([]string, error) {
	req := fmt.Sprintf(`{"keepalive":true, "request":"addpeer", "uri":"%s"}`, uri)
	resp, err := y.execReq(req)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(resp.Response))
	added := struct {
		Added []string `json:"added"`
	}{}
	if err := json.Unmarshal(resp.Response, &added); err != nil {
		return nil, err
	}

	return added.Added, nil
}
