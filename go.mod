module github.com/threefoldtech/zos

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff/v3 v3.0.0
	github.com/containerd/cgroups v0.0.0-20190911145653-fc51bcbe4714 // indirect
	github.com/containerd/containerd v1.3.2
	github.com/containerd/continuity v0.0.0-20190827140505-75bee3e2ccb6 // indirect
	github.com/containerd/fifo v0.0.0-20191213151349-ff969a566b00 // indirect
	github.com/containerd/ttrpc v0.0.0-20191028202541-4f1b8fe65a5c // indirect
	github.com/containerd/typeurl v0.0.0-20190911142611-5eb25027c9fd // indirect
	github.com/containernetworking/cni v0.7.2-0.20190807151350-8c6c47d1c7fc
	github.com/containernetworking/plugins v0.8.4
	github.com/dave/jennifer v1.3.0
	github.com/deckarep/golang-set v1.7.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/emicklei/dot v0.10.1
	github.com/firecracker-microvm/firecracker-go-sdk v0.19.1-0.20200110212531-741fc8cb0f2e
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gizak/termui/v3 v3.1.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-openapi/analysis v0.19.7 // indirect
	github.com/go-openapi/errors v0.19.3 // indirect
	github.com/go-openapi/runtime v0.19.9 // indirect
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/go-openapi/strfmt v0.19.4 // indirect
	github.com/go-openapi/swag v0.19.6 // indirect
	github.com/go-openapi/validate v0.19.5 // indirect
	github.com/go-redis/redis v6.15.5+incompatible
	github.com/gogo/googleapis v1.3.0 // indirect
	github.com/gomodule/redigo v1.7.0
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/iancoleman/strcase v0.0.0-20190422225806-e506e3ef7365
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/opencontainers/runtime-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.17.2
	github.com/shirou/gopsutil v2.19.11+incompatible
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/threefoldtech/zbus v0.1.2
	github.com/urfave/cli v1.22.1
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/whs/nacl-sealed-box v0.0.0-20180930164530-92b9ba845d8d
	go.etcd.io/bbolt v1.3.3 // indirect
	go.mongodb.org/mongo-driver v1.2.1 // indirect
	golang.org/x/crypto v0.0.0-20200109152110-61a87790db17
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20200113162924-86b910548bc1
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20191219145116-fa6499c8e75f
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/grpc v1.24.0 // indirect
	gopkg.in/yaml.v2 v2.2.7
	gotest.tools v2.2.0+incompatible
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
