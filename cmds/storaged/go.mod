module github.com/threefoldtech/zosv2/cmds/storaged

go 1.13

require (
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/rs/zerolog v1.15.0
	github.com/threefoldtech/zbus v0.0.0-20190711124326-09379d5f12e0
	github.com/threefoldtech/zosv2/modules v0.0.0-20190617092247-12858dd3a7ac
)

replace github.com/threefoldtech/zosv2/modules => ../../modules
