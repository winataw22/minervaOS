module github.com/threefoldtech/zos

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff/v3 v3.0.0
	github.com/containerd/cgroups v0.0.0-20200327175542-b44481373989
	github.com/containerd/containerd v1.3.2
	github.com/containerd/typeurl v0.0.0-20190911142611-5eb25027c9fd
	github.com/containernetworking/cni v0.7.2-0.20190807151350-8c6c47d1c7fc
	github.com/containernetworking/plugins v0.8.4
	github.com/deckarep/golang-set v1.7.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/firecracker-microvm/firecracker-go-sdk v0.19.1-0.20200110212531-741fc8cb0f2e
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gizak/termui/v3 v3.1.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-redis/redis v6.15.5+incompatible
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/google/uuid v1.1.1
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/opencontainers/runtime-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.18.0
	github.com/shirou/gopsutil v2.19.11+incompatible
	github.com/stretchr/testify v1.5.1
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	github.com/termie/go-shutil v0.0.0-20140729215957-bcacb06fecae
	github.com/threefoldtech/tfexplorer v0.3.1-0.20200529110634-d262eb4cf6a0
	github.com/threefoldtech/zbus v0.1.3
	github.com/urfave/cli v1.22.3
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df
	github.com/whs/nacl-sealed-box v0.0.0-20180930164530-92b9ba845d8d
	go.etcd.io/bbolt v1.3.4 // indirect
	golang.org/x/crypto v0.0.0-20200311171314-f7b00557c8c4
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20191219145116-fa6499c8e75f
	gopkg.in/yaml.v2 v2.2.7
	gotest.tools v2.2.0+incompatible
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
