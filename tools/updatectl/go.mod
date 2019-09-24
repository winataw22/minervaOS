module github.com/threefoldtech/zosv2/tools/updatectl

go 1.13

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.15.0
	github.com/stretchr/testify v1.3.0
	github.com/threefoldtech/zosv2/modules v0.0.0-20190722144152-30e6baee90a7
	github.com/urfave/cli v1.20.0
)
