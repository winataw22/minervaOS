module github.com/threefoldtech/zosv2/cmds/networkd

go 1.12

require (
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c // indirect
	github.com/juju/errors v0.0.0-20190207033735-e65537c515d7 // indirect
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/utils v0.0.0-20180820210520-bf9cc5bdd62d // indirect
	github.com/rs/zerolog v1.14.3
	github.com/threefoldtech/zbus v0.0.0-20190711124326-09379d5f12e0
	github.com/threefoldtech/zosv2/modules v0.0.0-20190614135932-35b94bfa4dbe
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
	golang.org/x/sys v0.0.0-20190710143415-6ec70d6a5542 // indirect
	gopkg.in/juju/names.v2 v2.0.0-20180621093930-fd59336b4621 // indirect
)

replace github.com/threefoldtech/zosv2/modules => ../../modules/
