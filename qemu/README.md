# Run 0-OS in a VM using qemu

This folder contains a script that you can use to run 0-OS in a VM using qemu.

For ease of use a Makefile is also provided. To prepare your environment call `make prepare`. This will :

- copy `zinit` binary into the overlay
- download a 0-OS kernel

You also need to create a bridge called `zos0` and have a dhcp server giving out IP on the bridge range.

To start the 0-OS VM, do `make start`

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
- Set an ip address to this bridge:
```
ip a add 10.244.0.254/24 dev zos0
```
- Set bridge up:
```
ip l set dev zos0 up
```
- Ensure you can route internet traffic:
```
echo 1 > /proc/sys/net/ipv4/ip_forward
```
- Set your host as DNAT router (replace eth0 with your internet interface):
```
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```
- Ensure your firewall allows forwarding packets:
```
iptables -P FORWARD ACCEPT
```
- Start dnsmasq:
```
dnsmasq --interface zos0 --no-daemon --dhcp-range=10.244.0.100,10.244.0.200,2h
```

If you are using netctl to configure you bridge, here is an example config:

```
Description="zos0"
Interface=zos0
Connection=bridge
IP=static
Address=10.244.0.254/24 # use a range that fits your network
DNS=(8.8.8.8)
```

Example dnsmasq config:

```
interface=zos0
dhcp-range=10.244.0.100,10.244.0.244,2h
```

You can now start your virtual machine !

