module github.com/threefoldtech/zosv2/cmds/tnodb_mock

go 1.12

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/dspinhirne/netaddr-go v0.0.0-20180510133009-a6cfb692cb10
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.2
	github.com/stretchr/testify v1.3.0
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
)
