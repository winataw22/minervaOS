module github.com/threefoldtech/zosv2/cmds/bcdb_mock

go 1.13

replace github.com/threefoldtech/zosv2/modules => ../../modules/

require (
	github.com/dspinhirne/netaddr-go v0.0.0-20180510133009-a6cfb692cb10
	github.com/gorilla/handlers v1.4.1
	github.com/gorilla/mux v1.7.2
	github.com/stretchr/testify v1.3.0
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
)
