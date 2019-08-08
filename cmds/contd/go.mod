module github.com/threefoldtech/zosv2/cmds/contd

go 1.12

require (
	github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a // indirect
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/rs/zerolog v1.14.3
	github.com/threefoldtech/zbus v0.0.0-20190711124326-09379d5f12e0
	github.com/threefoldtech/zosv2/modules v0.0.0-20190614135932-35b94bfa4dbe
)

replace github.com/threefoldtech/zosv2/modules => ../../modules/
