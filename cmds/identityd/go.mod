module github.com/threefoldtech/zosv2/cmds/identityd

go 1.13

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/cenkalti/backoff/v3 v3.0.0
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.15.0
	github.com/sirupsen/logrus v1.4.0 // indirect
	github.com/threefoldtech/zbus v0.0.0-20190711124326-09379d5f12e0
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
)
