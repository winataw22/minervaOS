module github.com/threefoldtech/zosv2/cmds/storaged

go 1.12

require (
	github.com/rs/zerolog v1.14.3
	github.com/sirupsen/logrus v1.4.2
	github.com/threefoldtech/zbus v0.0.0-20190613083559-f8f4719d6595
	github.com/threefoldtech/zosv2/modules v0.0.0-20190617092247-12858dd3a7ac
)

replace github.com/threefoldtech/zosv2/modules => ../../modules
