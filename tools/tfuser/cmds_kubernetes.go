package main

import (
	"encoding/hex"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/urfave/cli"
)

func generateKubernetes(c *cli.Context) error {
	var (
		size            = c.Uint("size")
		netID           = c.String("network-id")
		ipString        = c.String("ip")
		plainSecret     = c.String("secret")
		nodeID          = c.String("node")
		masterIPStrings = c.StringSlice("master-ips")
		sshKeys         = c.StringSlice("ssh-keys")
	)

	if size == 0 || size > 2 {
		return errors.New("only size 1 or 2 is supported for vm")
	}

	if netID == "" {
		return errors.New("vm requires a network to run in")
	}

	ip := net.ParseIP(ipString)
	if ip.To4() == nil {
		return errors.New("bad IP for vm")
	}

	if plainSecret == "" {
		return errors.New("a secret is required for kubernetes")
	}

	pk, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		return errors.Wrap(err, "failed to parse nodeID")
	}

	encrypted, err := crypto.Encrypt([]byte(plainSecret), pk)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt private key")
	}
	encryptedSecret := hex.EncodeToString(encrypted)

	masterIPs := make([]net.IP, len(masterIPStrings))
	for i, mips := range masterIPStrings {
		mip := net.ParseIP(mips)
		if mip.To4() == nil {
			return errors.New("bad master IP for vm")
		}
		masterIPs[i] = mip
	}

	kube := provision.Kubernetes{
		Size:          uint8(size),
		NetworkID:     pkg.NetID(netID),
		IP:            ip,
		ClusterSecret: encryptedSecret,
		MasterIPs:     masterIPs,
		SSHKeys:       sshKeys,
	}

	p, err := embed(kube, provision.KubernetesReservation, c.String("node"))
	if err != nil {
		return errors.Wrap(err, "could not generate reservation schema")
	}

	return writeWorkload(c.GlobalString("schema"), p)
}
