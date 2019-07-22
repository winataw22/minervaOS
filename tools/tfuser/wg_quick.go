package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"strings"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/crypto"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/network/ip"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func genWGQuick(network *modules.Network, userID string) (string, error) {

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

	key, err := extractPrivateKey(localNr)
	if err != nil {
		return "", err
	}

	d := data{PrivateKey: key.String()}

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

func extractPrivateKey(r *modules.NetResource) (wgtypes.Key, error) {
	key := wgtypes.Key{}

	peer := getPeer(r.Peers, r.Prefix.String())

	if peer.Connection.PrivateKey == "" {
		return key, fmt.Errorf("wireguard private key is empty")
	}

	// private key is hex encoded in the network object
	sk := ""
	_, err := fmt.Sscanf(peer.Connection.PrivateKey, "%x", &sk)
	if err != nil {
		return key, err
	}

	// TODO: change me once identity is available over zbus
	keyPair, err := identity.LoadSeed("/var/cache/seed.txt")
	if err != nil {
		return key, err
	}
	decoded, err := crypto.Decrypt([]byte(sk), keyPair.PrivateKey)
	if err != nil {
		return key, err
	}

	return wgtypes.ParseKey(string(decoded))
}
