module github.com/threefoldtech/zosv2/cmds/networkd

go 1.12

require (
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/rs/zerolog v1.14.3
	github.com/threefoldtech/zbus v0.0.0-20190613083559-f8f4719d6595
	github.com/threefoldtech/zosv2/modules v0.0.0-20190614135932-35b94bfa4dbe
	github.com/vishvananda/netlink v1.0.0
)

replace github.com/threefoldtech/zosv2/modules => ../../modules/
