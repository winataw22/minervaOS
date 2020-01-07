package provision

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/threefoldtech/zos/pkg"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/provision"
	schema "github.com/threefoldtech/zos/pkg/schema"
	"gotest.tools/assert"
)

func TestEnum(t *testing.T) {
	r := TfgridReservationWorkload1{
		Type: TfgridReservationWorkload1TypeContainer,
	}

	bytes, err := json.Marshal(r)
	require.NoError(t, err)

	var o TfgridReservationWorkload1

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
		Volumes           []TfgridReservationContainerMount1
		NetworkConnection []TfgridReservationNetworkConnection1
		StatsAggregator   []TfgridReservationStatsaggregator1
	}
	tests := []struct {
		name    string
		fields  fields
		want    provision.Container
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
			want: provision.Container{
				FList:        "https://hub.grid.tf/tf-official-apps/ubuntu-bionic-build.flist",
				FlistStorage: "zdb://hub.grid.tf:9900",
				Env:          map[string]string{"FOO": "BAR"},
				Entrypoint:   "/sbin/my_init",
				Interactive:  false,
				Mounts:       []provision.Mount{},
				Network:      provision.Network{},
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
				Volumes: []TfgridReservationContainerMount1{
					{
						VolumeID:   "volume1",
						Mountpoint: "/mnt",
					},
					{
						VolumeID:   "volume2",
						Mountpoint: "/data",
					},
				},
				NetworkConnection: []TfgridReservationNetworkConnection1{
					{
						NetworkID: "net1",
						Ipaddress: net.ParseIP("10.0.0.1"),
					},
				},
				StatsAggregator: nil,
			},
			want: provision.Container{
				FList:        "https://hub.grid.tf/tf-official-apps/ubuntu-bionic-build.flist",
				FlistStorage: "zdb://hub.grid.tf:9900",
				Env:          map[string]string{"FOO": "BAR"},
				Entrypoint:   "/sbin/my_init",
				Interactive:  false,
				Mounts: []provision.Mount{
					{
						VolumeID:   "volume1",
						Mountpoint: "/mnt",
					},
					{
						VolumeID:   "volume2",
						Mountpoint: "/data",
					},
				},
				Network: provision.Network{
					NetworkID: "net1",
					IPs:       []net.IP{net.ParseIP("10.0.0.1")},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := TfgridReservationContainer1{
				WorkloadID:        tt.fields.WorkloadID,
				NodeID:            tt.fields.NodeID,
				Flist:             tt.fields.Flist,
				HubURL:            tt.fields.HubURL,
				Environment:       tt.fields.Environment,
				Entrypoint:        tt.fields.Entrypoint,
				Interactive:       tt.fields.Interactive,
				Volumes:           tt.fields.Volumes,
				NetworkConnection: tt.fields.NetworkConnection,
				StatsAggregator:   tt.fields.StatsAggregator,
			}
			got, _, err := c.ToProvisionType()
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
		Type            TfgridReservationVolume1TypeEnum
		StatsAggregator []TfgridReservationStatsaggregator1
	}
	tests := []struct {
		name    string
		fields  fields
		want    provision.Volume
		wantErr bool
	}{
		{
			name: "HDD",
			fields: fields{
				WorkloadID:      1,
				NodeID:          "node1",
				Size:            10,
				Type:            TfgridReservationVolume1TypeHDD,
				StatsAggregator: nil,
			},
			want: provision.Volume{
				Size: 10,
				Type: provision.HDDDiskType,
			},
		},
		{
			name: "SSD",
			fields: fields{
				WorkloadID:      1,
				NodeID:          "node1",
				Size:            10,
				Type:            TfgridReservationVolume1TypeSSD,
				StatsAggregator: nil,
			},
			want: provision.Volume{
				Size: 10,
				Type: provision.SSDDiskType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := TfgridReservationVolume1{
				WorkloadID:      tt.fields.WorkloadID,
				NodeID:          tt.fields.NodeID,
				Size:            tt.fields.Size,
				Type:            tt.fields.Type,
				StatsAggregator: tt.fields.StatsAggregator,
			}
			got, _, err := v.ToProvisionType()
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
		Mode          TfgridReservationZdb1ModeEnum
		Password      string
		DiskType      TfgridReservationZdb1DiskTypeEnum
		Public        bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    provision.ZDB
		wantErr bool
	}{
		{
			name: "seq hdd",
			fields: fields{
				WorkloadID: 1,
				NodeID:     "node1",
				// ReservationID:,
				Size:     10,
				Mode:     TfgridReservationZdb1ModeSeq,
				Password: "supersecret",
				DiskType: TfgridReservationZdb1DiskTypeHdd,
				Public:   true,
			},
			want: provision.ZDB{
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
				Mode:     TfgridReservationZdb1ModeUser,
				Password: "supersecret",
				DiskType: TfgridReservationZdb1DiskTypeHdd,
				Public:   true,
			},
			want: provision.ZDB{
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
				Mode:     TfgridReservationZdb1ModeUser,
				Password: "supersecret",
				DiskType: TfgridReservationZdb1DiskTypeSsd,
				Public:   true,
			},
			want: provision.ZDB{
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
			z := TfgridReservationZdb1{
				WorkloadID:    tt.fields.WorkloadID,
				NodeID:        tt.fields.NodeID,
				ReservationID: tt.fields.ReservationID,
				Size:          tt.fields.Size,
				Mode:          tt.fields.Mode,
				Password:      tt.fields.Password,
				DiskType:      tt.fields.DiskType,
				Public:        tt.fields.Public,
			}
			got, _, err := z.ToProvisionType()
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
		StatsAggregator  []TfgridReservationStatsaggregator1
		NetworkResources []TfgridNetworkNetResource1
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
			n := TfgridReservationNetwork1{
				Name:             tt.fields.Name,
				WorkloadID:       tt.fields.WorkloadID,
				Iprange:          tt.fields.Iprange,
				StatsAggregator:  tt.fields.StatsAggregator,
				NetworkResources: tt.fields.NetworkResources,
			}
			got, err := n.ToProvisionType()
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
		Peers                        []WireguardPeer1
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
			r := TfgridNetworkNetResource1{
				NodeID:                       tt.fields.NodeID,
				IPRange:                      tt.fields.IPRange,
				WireguardPrivateKeyEncrypted: tt.fields.WireguardPrivateKeyEncrypted,
				WireguardPublicKey:           tt.fields.WireguardPublicKey,
				WireguardListenPort:          tt.fields.WireguardListenPort,
				Peers:                        tt.fields.Peers,
			}
			got, err := r.ToProvisionType()
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
		AllowedIPs []string
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
				PublicKey:  "0t11OkPwUBPe6m6wL6JTVzJHNjjReBJbEcnSZPs+pFo=",
				Endpoint:   "192.168.1.1",
				AllowedIPs: []string{"192.168.1.0/24", "172.20.0.0/16"},
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
				AllowedIPs: []string{"192.168.1.0"},
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
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := WireguardPeer1{
				PublicKey:  tt.fields.PublicKey,
				Endpoint:   tt.fields.Endpoint,
				AllowedIPs: tt.fields.AllowedIPs,
			}
			got, err := p.ToProvisionType()
			if !tt.wantErr {
				require.NoError(t, err)
				assert.DeepEqual(t, tt.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}
