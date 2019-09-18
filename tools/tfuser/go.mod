module github.com/threefoldtech/zosv2/cmds/tfuser

go 1.13

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/deckarep/golang-set v1.7.1
	github.com/google/uuid v1.1.1
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.15.0
	github.com/stretchr/testify v1.3.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
	github.com/urfave/cli v1.20.0
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20190607034155-226bf4e412cd
)
