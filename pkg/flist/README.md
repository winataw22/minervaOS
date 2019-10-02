# Flist module

## How to run the tests of this module

Because 0-fs requires root permission to mount the filesystems, you need to run the tests using the root user

An easy way to configure your system to have all your user GOPATH in the root directory is to create a symlink to `/root/go`.
Because if `$GOPATH` is not set, go automatically use `$HOME/go`

```shell
ln -s $GOPATH /root/go
```

Then to the tests:

```shell
su
go test -v
```

## RPC test

Because RPC tests requires to have a local redis running, RPC tests are not run by default.
In order to run the RPC test, you need to have redis running and listening on port 6379 and pass the `-rpc` flag to the `go test` command.

```shell
go test -v -rpc
```