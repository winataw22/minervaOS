module github.com/threefoldtech/zosv2/tools/updatectl

go 1.12

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/rs/zerolog v1.14.3
	github.com/threefoldtech/zosv2/modules v0.0.0-20190722144152-30e6baee90a7
	github.com/urfave/cli v1.20.0
)
