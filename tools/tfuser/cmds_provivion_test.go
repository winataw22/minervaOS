package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/provision"
)

func TestReservationSignature(t *testing.T) {
	schema := []byte(`{
		"id": "",
		"user_id": "Fuy6CatZGmmtWby13MESN2dYZUJi1ThjPR493w9TyWz1",
		"type": "container",
		"data": {
			"flist": "https://hub.grid.tf/zaibon/zaibon-ubuntu-ssh-0.0.2.flist",
			"flist_storage": "",
			"env": {},
			"entrypoint": "/sbin/my_init",
			"interactive": false,
			"mounts": [],
			"network": {
				"network_id": "zaibon_dev_network",
				"ips": [
					"172.22.4.20"
				]
			}
		},
		"created": "2019-10-10T10:18:46.704065596+02:00",
		"duration": 300000000000,
		"signature": "nJMRSZw7whyu6eTQ0sSEvS+RiHRRvk238MzsKeI6QFYQF4UGlivptcn9YN9uDm7C11a6GEJzli9Gr7YXE4dQDA==",
		"to_delete": false
	}`)
	keypair, err := identity.GenerateKeyPair()
	require.NoError(t, err)

	r := &provision.Reservation{}
	err = json.Unmarshal(schema, r)
	require.NoError(t, err)

	r.Duration = time.Second * 10
	r.Created = time.Now()
	r.User = keypair.Identity()

	err = r.Sign(keypair.PrivateKey)
	require.NoError(t, err)

	err = provision.Verify(r)
	assert.NoError(t, err)

	r = &provision.Reservation{}
	err = json.Unmarshal(schema, r)
	require.NoError(t, err)

	r.Duration = time.Second * 10
	r.Created = time.Now()
	r.User = keypair.Identity()

	err = r.Sign(keypair.PrivateKey)
	require.NoError(t, err)

	err = provision.Verify(r)
	assert.NoError(t, err)
}
