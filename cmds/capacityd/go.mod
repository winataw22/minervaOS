module github.com/threefoldtech/zosv2/cmds/capacityd

go 1.12

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.15.0
	github.com/threefoldtech/zbus v0.0.0-20190711124326-09379d5f12e0
	github.com/threefoldtech/zosv2/modules v0.0.0-20190826142459-a8c8b7c6e0b7
)

replace github.com/threefoldtech/zosv2/modules => ../../modules/
