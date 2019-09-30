module github.com/threefoldtech/zosv2/cmds/provisiond

go 1.13

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/mdlayher/genetlink v0.0.0-20190617154021-985b2115c31a // indirect
	github.com/mdlayher/netlink v0.0.0-20190617153422-f82a9b10b2bc // indirect
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.15.0
	github.com/threefoldtech/zbus v0.0.0-20190711124326-09379d5f12e0
	github.com/threefoldtech/zosv2/modules v0.0.0-20190711080824-231c81c6ccb4
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f // indirect
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20190629151639-28f4e240be2d // indirect
)
