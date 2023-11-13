module github.com/threefoldtech/zos/tools/zos-update-version

go 1.21

require (
	github.com/rs/zerolog v1.28.0
	github.com/spf13/cobra v1.6.1
	github.com/threefoldtech/substrate-client v0.1.5
)

require (
	github.com/ChainSafe/go-schnorrkel v1.0.0 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.5 // indirect
	github.com/cosmos/go-bip39 v1.0.0 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/base58 v1.0.3 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect
	github.com/ethereum/go-ethereum v1.10.17 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/gtank/merlin v0.1.1 // indirect
	github.com/gtank/ristretto255 v0.1.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mimoo/StrobeGo v0.0.0-20210601165009-122bf33a46e0 // indirect
	github.com/pierrec/xxHash v0.1.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/cors v1.8.2 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/vedhavyas/go-subkey v1.0.3 // indirect
	golang.org/x/crypto v0.0.0-20211117183948-ae814b36b871 // indirect
	golang.org/x/sys v0.13.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)

replace github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.5 => github.com/threefoldtech/go-substrate-rpc-client/v4 v4.0.6-0.20220927094755-0f0d22c73cc7
