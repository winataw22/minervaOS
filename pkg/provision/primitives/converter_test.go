package primitives

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/threefoldtech/zos/pkg"

	"github.com/stretchr/testify/require"
	schema "github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg/container/logger"
	"github.com/threefoldtech/zos/pkg/container/stats"
	"gotest.tools/assert"
)

func TestEnum(t *testing.T) {
	r := workloads.ReservationWorkload{
		Type: workloads.WorkloadTypeContainer,
	}

	bytes, err := json.Marshal(r)
	require.NoError(t, err)

	var o workloads.ReservationWorkload

	require.NoError(t, json.Unmarshal(bytes, &o))

	require.Equal(t, r.Type, o.Type)
}

func TestTfgridReservationContainer1_ToProvisionType(t *testing.T) {
	type fields struct {
		WorkloadID        int64
		NodeID            string
		Flist             string
		HubURL            string
		Environment       map[string]string
		Entrypoint        string
		Interactive       bool
		Volumes           []workloads.ContainerMount
		NetworkConnection []workloads.NetworkConnection
		StatsAggregator   []workloads.StatsAggregator
	}
	tests := []struct {
		name    string
		fields  fields
		want    Container
		wantErr bool
	}{
		{
			name: "empty network and volume",
			fields: fields{
				WorkloadID:        1,
				NodeID:            "node1",
				Flist:             "https://hub.grid.tf/tf-official-apps/ubuntu-bionic-build.flist",
				HubURL:            "zdb://hub.grid.tf:9900",
				Environment:       map[string]string{"FOO": "BAR"},
				Entrypoint:        "/sbin/my_init",
				Interactive:       false,
				Volumes:           nil,
				NetworkConnection: nil,
				StatsAggregator:   nil,
			},
			want: Container{
				FList:           "https://hub.grid.tf/tf-official-apps/ubuntu-bionic-build.flist",
				FlistStorage:    "zdb://hub.grid.tf:9900",
				Env:             map[string]string{"FOO": "BAR"},
				SecretEnv:       nil,
				Entrypoint:      "/sbin/my_init",
				Interactive:     false,
				Mounts:          []Mount{},
				Network:         Network{},
				Logs:            []logger.Logs{},
				StatsAggregator: []stats.Aggregator{},
			},
			wantErr: false,
		},
		{
			name: "with network and volumes",
			fields: fields{
				WorkloadID:  1,
				NodeID:      "node1",
				Flist:       "https://hub.grid.tf/tf-official-apps/ubuntu-bionic-build.flist",
				HubURL:      "zdb://hub.grid.tf:9900",
				Environment: map[string]string{"FOO": "BAR"},
				Entrypoint:  "/sbin/my_init",
				Interactive: false,
				Volumes: []workloads.ContainerMount{
					{
						VolumeId:   "-volume1",
						Mountpoint: "/mnt",
					},
					{
						VolumeId:   "volume2",
						Mountpoint: "/data",
					},
				},
				NetworkConnection: []workloads.NetworkConnection{
					{
						NetworkId: "net1",
						Ipaddress: net.ParseIP("10.0.0.1"),
					},
				},
			},
			want: Container{
				FList:        "https://hub.grid.tf/tf-official-apps/ubuntu-bionic-build.flist",
				FlistStorage: "zdb://hub.grid.tf:9900",
				Env:          map[string]string{"FOO": "BAR"},
				SecretEnv:    nil,
				Entrypoint:   "/sbin/my_init",
				Interactive:  false,
				Mounts: []Mount{
					{
						VolumeID:   "reservation-volume1",
						Mountpoint: "/mnt",
					},
					{
						VolumeID:   "volume2",
						Mountpoint: "/data",
					},
				},
				Network: Network{
					NetworkID: "net1",
					IPs:       []net.IP{net.ParseIP("10.0.0.1")},
				},
				Logs:            []logger.Logs{},
				StatsAggregator: []stats.Aggregator{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := workloads.Container{
				WorkloadId:        tt.fields.WorkloadID,
				NodeId:            tt.fields.NodeID,
				Flist:             tt.fields.Flist,
				HubUrl:            tt.fields.HubURL,
				Environment:       tt.fields.Environment,
				Entrypoint:        tt.fields.Entrypoint,
				Interactive:       tt.fields.Interactive,
				Volumes:           tt.fields.Volumes,
				NetworkConnection: tt.fields.NetworkConnection,
				StatsAggregator:   tt.fields.StatsAggregator,
			}
			got, _, err := ContainerToProvisionType(c, "reservation")
			if !tt.wantErr {
				require.NoError(t, err)
				assert.DeepEqual(t, tt.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestTfgridReservationVolume1_ToProvisionType(t *testing.T) {
	type fields struct {
		WorkloadID      int64
		NodeID          string
		ReservationID   int64
		Size            int64
		Type            workloads.VolumeTypeEnum
		StatsAggregator []workloads.StatsAggregator
	}
	tests := []struct {
		name    string
		fields  fields
		want    Volume
		wantErr bool
	}{
		{
			name: "HDD",
			fields: fields{
				WorkloadID:      1,
				NodeID:          "node1",
				Size:            10,
				Type:            workloads.VolumeTypeHDD,
				StatsAggregator: nil,
			},
			want: Volume{
				Size: 10,
				Type: pkg.HDDDevice,
			},
		},
		{
			name: "SSD",
			fields: fields{
				WorkloadID:      1,
				NodeID:          "node1",
				Size:            10,
				Type:            workloads.VolumeTypeSSD,
				StatsAggregator: nil,
			},
			want: Volume{
				Size: 10,
				Type: pkg.SSDDevice,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := workloads.Volume{
				WorkloadId:      tt.fields.WorkloadID,
				NodeId:          tt.fields.NodeID,
				Size:            tt.fields.Size,
				Type:            tt.fields.Type,
				StatsAggregator: tt.fields.StatsAggregator,
			}
			got, _, err := VolumeToProvisionType(v)
			if !tt.wantErr {
				require.NoError(t, err)
				assert.DeepEqual(t, tt.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestTfgridReservationZdb1_ToProvisionType(t *testing.T) {
	type fields struct {
		WorkloadID    int64
		NodeID        string
		ReservationID int64
		Size          int64
		Mode          workloads.ZDBModeEnum
		Password      string
		DiskType      workloads.DiskTypeEnum
		Public        bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    ZDB
		wantErr bool
	}{
		{
			name: "seq hdd",
			fields: fields{
				WorkloadID: 1,
				NodeID:     "node1",
				// ReservationID:,
				Size:     10,
				Mode:     workloads.ZDBModeSeq,
				Password: "supersecret",
				DiskType: workloads.DiskTypeHDD,
				Public:   true,
			},
			want: ZDB{
				Size:     10,
				Mode:     pkg.ZDBModeSeq,
				Password: "supersecret",
				DiskType: pkg.HDDDevice,
				Public:   true,
			},
			wantErr: false,
		},
		{
			name: "user hdd",
			fields: fields{
				WorkloadID: 1,
				NodeID:     "node1",
				// ReservationID:,
				Size:     10,
				Mode:     workloads.ZDBModeUser,
				Password: "supersecret",
				DiskType: workloads.DiskTypeHDD,
				Public:   true,
			},
			want: ZDB{
				Size:     10,
				Mode:     pkg.ZDBModeUser,
				Password: "supersecret",
				DiskType: pkg.HDDDevice,
				Public:   true,
			},
			wantErr: false,
		},
		{
			name: "user ssd",
			fields: fields{
				WorkloadID: 1,
				NodeID:     "node1",
				// ReservationID:,
				Size:     10,
				Mode:     workloads.ZDBModeUser,
				Password: "supersecret",
				DiskType: workloads.DiskTypeSSD,
				Public:   true,
			},
			want: ZDB{
				Size:     10,
				Mode:     pkg.ZDBModeUser,
				Password: "supersecret",
				DiskType: pkg.SSDDevice,
				Public:   true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := workloads.ZDB{
				WorkloadId: tt.fields.WorkloadID,
				NodeId:     tt.fields.NodeID,
				//ReservationID: tt.fields.ReservationID,
				Size:     tt.fields.Size,
				Mode:     tt.fields.Mode,
				Password: tt.fields.Password,
				DiskType: tt.fields.DiskType,
				Public:   tt.fields.Public,
			}
			got, _, err := ZDBToProvisionType(z)
			if !tt.wantErr {
				require.NoError(t, err)
				assert.DeepEqual(t, tt.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestTfgridReservationNetwork1_ToProvisionType(t *testing.T) {
	type fields struct {
		Name             string
		WorkloadID       int64
		Iprange          schema.IPRange
		StatsAggregator  []workloads.StatsAggregator
		NetworkResources []workloads.NetworkNetResource
	}
	tests := []struct {
		name    string
		fields  fields
		want    pkg.Network
		wantErr bool
	}{
		{
			name: "main",
			fields: fields{
				Name:             "net1",
				WorkloadID:       1,
				Iprange:          schema.MustParseIPRange("192.168.0.0/16"),
				NetworkResources: nil,
			},
			want: pkg.Network{
				Name:         "net1",
				NetID:        pkg.NetID("net1"),
				IPRange:      types.MustParseIPNet("192.168.0.0/16"),
				NetResources: []pkg.NetResource{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := workloads.Network{
				Name:             tt.fields.Name,
				WorkloadId:       tt.fields.WorkloadID,
				Iprange:          tt.fields.Iprange,
				StatsAggregator:  tt.fields.StatsAggregator,
				NetworkResources: tt.fields.NetworkResources,
			}
			got, err := NetworkToProvisionType(n)
			if !tt.wantErr {
				require.NoError(t, err)
				assert.DeepEqual(t, tt.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestTfgridNetworkNetResource1_ToProvisionType(t *testing.T) {
	type fields struct {
		NodeID                       string
		IPRange                      schema.IPRange
		WireguardPrivateKeyEncrypted string
		WireguardPublicKey           string
		WireguardListenPort          int64
		Peers                        []workloads.WireguardPeer
	}
	tests := []struct {
		name    string
		fields  fields
		want    pkg.NetResource
		wantErr bool
	}{
		{
			name: "main",
			fields: fields{
				NodeID:                       "node1",
				IPRange:                      schema.MustParseIPRange("192.168.0.0/16"),
				WireguardPrivateKeyEncrypted: "6C6C6568726F776FA646C",
				WireguardPublicKey:           "0t11OkPwUBPe6m6wL6JTVzJHNjjReBJbEcnSZPs+pFo=",
				WireguardListenPort:          6380,
			},
			want: pkg.NetResource{
				NodeID:       "node1",
				Subnet:       types.MustParseIPNet("192.168.0.0/16"),
				WGPrivateKey: "6C6C6568726F776FA646C",
				WGPublicKey:  "0t11OkPwUBPe6m6wL6JTVzJHNjjReBJbEcnSZPs+pFo=",
				WGListenPort: 6380,
				Peers:        []pkg.Peer{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := workloads.NetworkNetResource{
				NodeId:                       tt.fields.NodeID,
				Iprange:                      tt.fields.IPRange,
				WireguardPrivateKeyEncrypted: tt.fields.WireguardPrivateKeyEncrypted,
				WireguardPublicKey:           tt.fields.WireguardPublicKey,
				WireguardListenPort:          tt.fields.WireguardListenPort,
				Peers:                        tt.fields.Peers,
			}
			got, err := NetResourceToProvisionType(r)
			if !tt.wantErr {
				require.NoError(t, err)
				assert.DeepEqual(t, tt.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestWireguardPeer1_ToProvisionType(t *testing.T) {
	type fields struct {
		PublicKey  string
		Endpoint   string
		AllowedIPs []schema.IPRange
	}
	tests := []struct {
		name    string
		fields  fields
		want    pkg.Peer
		wantErr bool
	}{
		{
			name: "main",
			fields: fields{
				PublicKey: "0t11OkPwUBPe6m6wL6JTVzJHNjjReBJbEcnSZPs+pFo=",
				Endpoint:  "192.168.1.1",
				AllowedIPs: []schema.IPRange{
					schema.MustParseIPRange("192.168.1.0/24"),
					schema.MustParseIPRange("172.20.0.0/16"),
				},
			},
			want: pkg.Peer{
				// Subnet: types.ParseIPNet("")
				WGPublicKey: "0t11OkPwUBPe6m6wL6JTVzJHNjjReBJbEcnSZPs+pFo=",
				AllowedIPs: []types.IPNet{
					types.MustParseIPNet("192.168.1.0/24"),
					types.MustParseIPNet("172.20.0.0/16"),
				},
				Endpoint: "192.168.1.1",
			},
			wantErr: false,
		},
		{
			name: "wrong allowed IP",
			fields: fields{
				PublicKey:  "0t11OkPwUBPe6m6wL6JTVzJHNjjReBJbEcnSZPs+pFo=",
				Endpoint:   "192.168.1.1",
				AllowedIPs: []schema.IPRange{schema.MustParseIPRange("192.168.1.0/24")},
			},
			want: pkg.Peer{
				// Subnet: types.ParseIPNet("")
				WGPublicKey: "0t11OkPwUBPe6m6wL6JTVzJHNjjReBJbEcnSZPs+pFo=",
				AllowedIPs: []types.IPNet{
					types.MustParseIPNet("192.168.1.0/24"),
				},
				Endpoint: "192.168.1.1",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := workloads.WireguardPeer{
				PublicKey:      tt.fields.PublicKey,
				Endpoint:       tt.fields.Endpoint,
				AllowedIprange: tt.fields.AllowedIPs,
			}
			got, err := WireguardToProvisionType(p)
			if !tt.wantErr {
				require.NoError(t, err)
				assert.DeepEqual(t, tt.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}
