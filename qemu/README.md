# Run 0-OS in a VM using qemu

This folder contains a script that you can use to run 0-OS in a VM using qemu.

## Requirements

When you download a pre-built kernel, you're ready to go already, all you need
is able to run a qemu machine with an internet access (IPv6 is required).

If you want to hack the code and try in a real environment, you can use helpers provided here
to setup an overlay in qemu which (kind of) patch the kernel image on boot time.

### Default Image

You can simply use the makefile provided and do: `make kernel`

This will download the latest prebuilt kernel from our server.

### Hack the code

In order to use the full power of theses scripts, you'll need in addition:

- A self compiled version of [z-init](https://github.com/threefoldtech/zinit/)
- All the binaries from this repository compiled
- A working qemu with ipv6 internet reachability

Then you can use the full power of the overlay. Everything is already setup using symlinks
to use your local binaries inside the image.

### Use the Makefile

To prepare your environment call `make prepare`. This will:

- Copy `zinit` binary into the overlay
- Download a 0-OS kernel

To start the 0-OS VM, do `make start`

You can specify your farm id (if you have one) in the command line with:
```
make FARMERID=A3y5F8CoHVZiq3Sxxxx start
```

By default, by running `make start` you will start the VM and joining the development network
(uing the mock, not the bcdb network). To use the testing network and bcdb, you can use `make test`

### Reach internet inside the VM

You need to be able to reach internet inside the VM, 0-OS needs internet. The system will starts
even if you only have ipv4 address, but you will be quickly blocked because lot of deployment
features uses ipv6.

If you want to use theses script, you'll need to create a bridge called zos0 and
have a dhcp server giving out ip on the bridge range.

## Prepare the bridge manually

### QEMU Configuration

First thing to do, ensure your qemu's bridge configuration allows our bridge, edit file `/etc/qemu/bridge.conf`.
The bridge we will use is called `zos0`, you need to allow this bridge:

```
# This should have the following permissions: root:qemu 0640
# [...]
allow zos0
```

### Build our Bridge

- Next step, just create a new empty bridge:
```
brctl addbr zos0
```
- Move your local interface which have internet access into the bridge (let's say `eth0`):
```
brctl addif zos0 eth0
```
- Set bridge up:
```
ip l set dev zos0 up
```
- Ensure you can route internet traffic:
```
echo 1 > /proc/sys/net/ipv4/ip_forward
```
- Ensure your firewall allows forwarding packets:
```
iptables -P FORWARD ACCEPT
```

Now, inside the bridge, the VM will be connected to the network like it's another
computer, using the same network cable. You need to have a working DHCP server and Router Advertissment
behind to get IPv4 and IPv6. If it works on your host, it should works on the VM out of box.

You can now start your virtual machine !

