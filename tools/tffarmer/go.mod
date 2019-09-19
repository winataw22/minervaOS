module github.com/threefoldtech/zosv2/cmds/tffarmer

go 1.13

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/rs/zerolog v1.15.0
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
	github.com/urfave/cli v1.20.0
)
