package gedis

import (
	"fmt"
	"testing"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClient(t *testing.T) *Gedis {
	gedis, err := New("tcp://172.17.0.2:8901", "default", "")
	require.NoError(t, err)

	return gedis
}

func TestPing(t *testing.T) {
	gedis := testClient(t)

	resp, err := gedis.Ping()
	require.NoError(t, err)
	assert.Equal(t, "PONG", resp)
}

func TestNodesCRUD(t *testing.T) {
	gedis := testClient(t)

	id, err := gedis.RegisterNode(modules.StrIdentifier("node1"), modules.StrIdentifier("farm1"), "2.0.0")
	require.NoError(t, err)

	node, err := gedis.GetNode(modules.StrIdentifier(id))
	require.NoError(t, err)

	assert.Equal(t, "node1", node.NodeID)
	assert.Equal(t, "farm1", node.FarmID)

	err = gedis.UpdateTotalNodeCapacity(modules.StrIdentifier("node1"), 14, 37, 42, 420)
	require.NoError(t, err)

	err = gedis.UpdateReservedNodeCapacity(modules.StrIdentifier("node1"), 88, 44, 22, 11)
	require.NoError(t, err)

	err = gedis.UpdateUsedNodeCapacity(modules.StrIdentifier("node1"), 1, 2, 3, 4)
	require.NoError(t, err)

	err = gedis.ListFarm("", "")
	err = gedis.ListNode(modules.StrIdentifier("farm1"), "", "")

	addrs := []string{"w411et_1"}
	// FIXME: overwrite fails
	fid, _ := gedis.RegisterFarm(modules.StrIdentifier("farm1"), "My Debug Farm 4", "bot@farmer.tld", addrs)
	require.NoError(t, err)
	fmt.Println(fid)

	/*
		f, err := gedis.GetFarm(modules.StrIdentifier("My Debug Farm 4"))
		require.NoError(t, err)

		assert.Equal(t, "farm1", f.Name)
	*/
}

// func TestListNode(t *testing.T) {
// 	gedis, err := New("tcp://172.17.0.2:8900", "default", "")
// 	require.NoError(t, err)

// 	err = gedis.Connect()
// 	require.NoError(t, err)

// 	nodes, err := gedis.ListNodes()
// 	require.NoError(t, err)
// 	fmt.Println(nodes)
// }

// func TestRegisterFarm(t *testing.T) {
// 	gedis, err := New("tcp://172.17.0.2:8900", "default", "")
// 	require.NoError(t, err)

// 	err = gedis.Connect()
// 	require.NoError(t, err)

// 	err = gedis.RegisterFarm(modules.StrIdentifier("farm1"), "my super farm")
// 	require.NoError(t, err)
// 	// id, err = gedis.RegisterNode(modules.StrIdentifier(fmt.Sprintf("%d", id)), modules.StrIdentifier("farm1"))
// 	// require.NoError(t, err)
// }
