module github.com/threefoldtech/zosv2/cmds/tffarmer

go 1.12

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/rs/zerolog v1.14.3
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
	github.com/urfave/cli v1.20.0
)
