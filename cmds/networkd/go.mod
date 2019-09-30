module github.com/threefoldtech/zosv2/cmds/networkd

go 1.13

require (
	github.com/cenkalti/backoff/v3 v3.0.0
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.15.0
	github.com/threefoldtech/zbus v0.0.0-20190711124326-09379d5f12e0
	github.com/threefoldtech/zosv2/modules v0.0.0-20190614135932-35b94bfa4dbe
	github.com/vishvananda/netlink v1.0.0
)

replace github.com/threefoldtech/zosv2/modules => ../../modules/
