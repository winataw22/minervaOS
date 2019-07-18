package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"os"
	"strings"

	"github.com/threefoldtech/zosv2/modules/network/ip"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/urfave/cli"
)

func createNetwork(c *cli.Context) error {
	farmID := c.String("farm")
	if farmID == "" {
		return fmt.Errorf("farm ID must be specified")
	}

	network, err := db.CreateNetwork(farmID)
	if err != nil {
		log.Error().Err(err).Msg("failed to create network")
		return err
	}
	fmt.Printf("network created: %s\n", network.NetID)

	n := provision.Network{
		NetwokID: string(network.NetID),
	}
	raw, err := json.Marshal(n)
	if err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r := provision.Reservation{
		ID:   id.String(),
		Type: provision.NetworkReservation,
		Data: raw,
	}

	nodeIDs := c.StringSlice("node")
	for _, nodeID := range nodeIDs {
		if err := store.Reserve(r, identity.StrIdentifier(nodeID)); err != nil {
			return err
		}
		fmt.Printf("network reservation sent for node ID %s\n", nodeID)
	}

	return nil
}

func addMember(c *cli.Context) error {
	nwID := c.String("network")
	if nwID == "" {
		return fmt.Errorf("network ID must be specified")
	}

	network, err := db.GetNetwork(modules.NetID(nwID))
	if err != nil {
		log.Error().Err(err).Msgf("network %s does not exists", nwID)
		return err
	}

	n := provision.Network{
		NetwokID: string(network.NetID),
	}
	raw, err := json.Marshal(n)
	if err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r := provision.Reservation{
		ID:   id.String(),
		Type: provision.NetworkReservation,
		Data: raw,
	}

	nodeIDs := c.StringSlice("node")
	for _, nodeID := range nodeIDs {
		if err := store.Reserve(r, identity.StrIdentifier(nodeID)); err != nil {
			return err
		}
		fmt.Printf("network reservation sent for node ID %s\n", nodeID)
	}

	return nil
}

func addLocal(c *cli.Context) error {
	nwID := c.String("network")
	if nwID == "" {
		return fmt.Errorf("network ID must be specified")
	}

	userID := c.String("user")
	if userID == "" {
		return fmt.Errorf("user ID must be specified")
	}

	network, err := db.GetNetwork(modules.NetID(nwID))
	if err != nil {
		log.Error().Err(err).Msgf("network %s does not exists", nwID)
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	key, err := wireguard.GenerateKey(cwd)
	if err != nil {
		return err
	}

	network, err = db.AddUser(
		identity.StrIdentifier(userID),
		modules.NetID(nwID),
		key.PublicKey().String())
	if err != nil {
		return err
	}

	var nr *modules.NetResource
	var peer *modules.Peer

	for _, r := range network.Resources {
		if r.NodeID.ID == userID {
			nr = r
			break
		}
	}
	if nr == nil {
		return fmt.Errorf("network doesn't container a network resource for your user. Something went wrong")
	}

	for _, p := range nr.Peers {
		if p.Prefix.String() == nr.Prefix.String() {
			peer = p
			break
		}
	}
	if peer == nil {
		return fmt.Errorf("network doesn't container a wireguard peer for your user. Something went wrong")
	}

	conf, err := genWGQuick(network, userID, key.String())
	if err != nil {
		return err
	}

	fmt.Printf("New network resource added\n")
	fmt.Printf("Here is your wg-quick config:\n\n")
	fmt.Println(conf)

	return nil
}

func genWGQuick(network *modules.Network, userID string, privateKey string) (string, error) {

	type Peer struct {
		Key        string
		AllowedIPs string
		Endpoint   string
		Port       uint16
	}
	type data struct {
		PrivateKey string
		Address    string
		Peer       Peer
	}

	localNr := getNetRes(network.Resources, userID)
	if localNr == nil {
		return "", fmt.Errorf("missing network resource for user %s", userID)
	}
	exitNr := getExitNetRes(network.Resources)
	if exitNr == nil {
		return "", fmt.Errorf("no exit network resource found in network")
	}
	exitPeer := getPeer(exitNr.Peers, exitNr.Prefix.String())
	if exitPeer == nil {
		return "", fmt.Errorf("missing exit peer %s", exitNr.Prefix.String())
	}

	d := data{PrivateKey: privateKey}

	localNibble := ip.NewNibble(localNr.Prefix, 0)
	a, b, err := localNibble.ToV4()
	if err != nil {
		return "", err
	}

	d.Address = strings.Join([]string{
		localNr.Prefix.String(),
		localNr.LinkLocal.String(),
		fmt.Sprintf("10.255.%d.%d/16", a, b),
	}, ", ")

	netIPNet := network.PrefixZero
	netIPNet.Mask = net.CIDRMask(48, 128)

	d.Peer = Peer{
		Key:      exitPeer.Connection.Key,
		Port:     exitPeer.Connection.Port,
		Endpoint: endpoint(exitPeer),
		AllowedIPs: strings.Join([]string{
			"fe80::1/128",
			"10.0.0.0/8",
			netIPNet.String(),
		}, ", "),
	}

	tmpl, err := template.New("wg").Parse(wgTmpl)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}

	if err := tmpl.Execute(buf, d); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var wgTmpl = `
[Interface]
PrivateKey = {{.PrivateKey}}
Address = {{.Address}}


[Peer]
PublicKey = {{.Peer.Key}}
AllowedIPs = {{.Peer.AllowedIPs}}
PersistentKeepalive = 20
{{if .Peer.Endpoint}}Endpoint = {{.Peer.Endpoint}}{{end}}
`

func getNetRes(nrs []*modules.NetResource, id string) *modules.NetResource {
	for _, nr := range nrs {
		if nr.NodeID.ID == id {
			return nr
		}
	}
	return nil
}

func getExitNetRes(nrs []*modules.NetResource) *modules.NetResource {
	for _, nr := range nrs {
		if nr.ExitPoint {
			return nr
		}
	}
	return nil
}

func getPeer(peers []*modules.Peer, prefix string) *modules.Peer {
	for _, p := range peers {
		if p.Prefix.String() == prefix {
			return p
		}
	}
	return nil
}

func endpoint(peer *modules.Peer) string {
	var endpoint string
	if peer.Connection.IP.To16() != nil {
		endpoint = fmt.Sprintf("[%s]:%d", peer.Connection.IP.String(), peer.Connection.Port)
	} else {
		endpoint = fmt.Sprintf("%s:%d", peer.Connection.IP.String(), peer.Connection.Port)
	}
	return endpoint
}
