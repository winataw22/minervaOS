module github.com/threefoldtech/zosv2/cmds/flistd

go 1.12

require (
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/testify v1.3.0
	github.com/threefoldtech/zbus v0.0.0-20190613083559-f8f4719d6595
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
)

replace github.com/threefoldtech/zosv2/modules => ../../modules/
