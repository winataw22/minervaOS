module github.com/threefoldtech/zosv2/cmds/tfuser

go 1.12

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/google/uuid v1.1.1
	github.com/minio/minio-go v6.0.14+incompatible // indirect
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/testify v1.3.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
	github.com/urfave/cli v1.20.0
)
