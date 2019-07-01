module github.com/threefoldtech/zosv2/cmds/contctl

go 1.12

require (
	github.com/containernetworking/plugins v0.8.1 // indirect
	github.com/rs/zerolog v1.14.3
	github.com/threefoldtech/zbus v0.0.0-20190613083559-f8f4719d6595
	github.com/threefoldtech/zosv2/modules v0.0.0-20190614135932-35b94bfa4dbe
	github.com/urfave/cli v1.20.0
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f // indirect
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20190614145803-89b2114fdddf // indirect
)

replace github.com/threefoldtech/zosv2/modules => ../../modules/
