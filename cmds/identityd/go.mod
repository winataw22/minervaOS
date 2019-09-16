module github.com/threefoldtech/zosv2/cmds/identityd

go 1.13

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/cenkalti/backoff/v3 v3.0.0
	github.com/rs/zerolog v1.14.3
	github.com/threefoldtech/zbus v0.0.0-20190613083559-f8f4719d6595
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
)
