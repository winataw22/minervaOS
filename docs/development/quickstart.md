# Quick start

- [Quick start](#quick-start)
  - [Starting a local zos node](#starting-a-local-zos-node)
  - [Accessing node](#accessing-node)
  - [Development](#development)
- [Qemu docs](../../qemu/README.md)

## Starting a local zos node

* Make sure `qemu` and `dnsmasq` are installed
* [Create a farm](../manual/manual.md#creating-a-farm)
* [Download a zos image](https://bootstrap.grid.tf/kernel/zero-os-development-zos-v3-generic-7e587e499a.efi)
* Make sure `zos0` bridge is allowed by qemu, you can add `allow zos0` in `/etc/qemu/bridge.conf` (create the file if it's not there)
* Setup the network using this script [this script](./net.sh)

Then, inside zos repository

```
make -C cmds
cd qemu
mv <downloaded image path> ./zos.efi
sudo ./vm.sh -n node-01 -c "farmer_id=<your farm id here> printk.devmsg=on runmode=dev"
```

You should see the qemu console and boot logs, wait for awhile and you can [browse farms](https://dashboard.dev.grid.tf/explorer/farms) to see your node is added/detected automatically.

To stop the machine you can do `Control + a` then `x`.

You can read more about setting up a qemu development environment and more network options [here](../../qemu/README.md).

## Accessing node

After booting up, the node should start downloading external packages, this would take some time depending on your internet connection.

See [how to ssh into it.](../../qemu/README.md#to-ssh-into-the-machine)

How to get the node IP?
Given the network script `dhcp-range`, it usually would be one of `192.168.123.43`, `192.168.123.44` or `192.168.123.45`. 

Or you can simply install `arp-scan` then do something like:

```
✗ sudo arp-scan --interface=zos0 --localnet
Interface: zos0, type: EN10MB, MAC: de:26:45:e6:87:95, IPv4: 192.168.123.1
Starting arp-scan 1.9.7 with 256 hosts (https://github.com/royhills/arp-scan)
192.168.123.44  54:43:83:1f:eb:81       (Unknown)
```

Now we know for sure it's `192.168.123.44`.

To check logs and see if the downloading of packages is still in progress, you can simply do:

```
zinit log
```

## Development

While the overlay will enable your to boot with the binaries that's been built locally, sometimes you'll need to test the changes of certain modules without restarting the node (or intending to do so for e.g. testing a migration).

For example if we changed anything related to `noded`, we can do the following:

Inside zos repository:

* Build binaries locally
    * `make -C cmds`
* Copy the binary inside the machine
    * `scp bin/zos root@192.168.123.44:/bin/noded`
* SSH into the machine then use `zinit` to restart it: 
    * `zinit stop noded && zinit start noded`
