module github.com/threefoldtech/zos

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/ChainSafe/go-schnorrkel v1.0.0
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/alexflint/go-filemutex v1.1.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/boltdb/bolt v1.3.1
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.2
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/containerd/cgroups v1.0.1
	github.com/containerd/containerd v1.5.0-rc.2
	github.com/containerd/continuity v0.2.0 // indirect
	github.com/containerd/typeurl v1.0.2
	github.com/containernetworking/cni v0.8.1
	github.com/containernetworking/plugins v0.9.1
	github.com/coreos/go-iptables v0.6.0 // indirect
	github.com/deckarep/golang-set v1.8.0
	github.com/decred/base58 v1.0.3
	github.com/diskfs/go-diskfs v1.2.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/g0rbe/go-chattr v0.0.0-20190906133247-aa435a6a0a37
	github.com/gizak/termui/v3 v3.1.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/gomodule/redigo v1.8.9
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/gtank/merlin v0.1.1
	github.com/hanwen/go-fuse/v2 v2.1.0 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6
	github.com/joncrlsn/dque v0.0.0-20200702023911-3e80e3146ce5
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/lestrrat-go/jwx v1.1.7
	github.com/mattn/go-sqlite3 v1.14.7 // indirect
	github.com/mdlayher/netlink v1.4.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/nsf/termbox-go v1.1.1 // indirect
	github.com/onsi/gomega v1.16.0 // indirect
	github.com/opencontainers/runc v1.0.2 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rs/zerolog v1.26.0
	github.com/rusart/muxprom v0.0.0-20200609120753-9173fa27435a
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/safchain/ethtool v0.0.0-20201023143004-874930cb3ce0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/threefoldtech/0-fs v1.3.1-0.20201203163303-d963de9adea7
	github.com/threefoldtech/go-rmb v0.1.11-0.20220224131627-825c23c921d3
	github.com/threefoldtech/substrate-client v0.0.0-20220629072807-544d41e5b81a
	github.com/threefoldtech/zbus v1.0.0
	github.com/tinylib/msgp v1.1.5 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/urfave/cli v1.22.5 // indirect
	github.com/urfave/cli/v2 v2.3.0
	github.com/vishvananda/netlink v1.1.1-0.20201029203352-d40f9887b852
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/whs/nacl-sealed-box v0.0.0-20180930164530-92b9ba845d8d
	github.com/yggdrasil-network/yggdrasil-go v0.4.0
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20211117183948-ae814b36b871
	golang.org/x/sys v0.0.0-20211124211545-fe61309f8881
	golang.zx2c4.com/wireguard v0.0.20200320 // indirect
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200609130330-bd2cb7843e1b
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210917145530-b395a37504d4 // indirect
	google.golang.org/grpc v1.41.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible

replace github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.2 => github.com/threefoldtech/go-substrate-rpc-client/v4 v4.0.3-0.20220629062423-919d421680f4
