# Run 0-OS in a VM using qemu

| For a quick development docs check [here](../docs/development/README.md)

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

### Build the Binaries

To build the binaries do `cd cmds && make`

### Use the Makefile

To prepare your environment call `make prepare`. This will:

- Copy `zinit` binary into the overlay
- Download a 0-OS kernel

### QEMU Configuration for the network

For qemu to run, you need the have the vm connected to a network.

There are 2 ways to do that, each with their own pitfalls.

- using a bridge, that hosts it's own network, and NATs that network with the IP of the host
  That means you have to provide your own IP-management on that network (dnsmasq), and you have to setup the NAT part.
- also with a bridge, but that has a real connection  interface as slave
  That way, your local network is providing for ip-configuration, and that will only work on wired interfaces. (i.e. wifi interfaces can't be a bridge slave)

It's important to make the distinction: in case of your own hosted network, that doesn't look like it would be a real ZOS host connected to a network, which basically means that that is better not used for network testing.

When using a direct wired connection, on a Linux development machine, you'll have to set it up on your own, as a personal computer has all sorts of automagic network setup tools.

First thing to do, ensure your qemu's bridge configuration allows our bridge, edit file `/etc/qemu/bridge.conf`.
The bridge we will use is called `zos0`, you need to allow this bridge:

```bash
# This should have the following permissions: root:qemu 0640
# [...]
allow zos0
```

## Case1, your own virtual natted network

1. first and foremost, create your bridge , and give it an IP address.

```bash
sudo ip link add zos0 type bridge
sudo ip link set zos0 up
```

As it's a separate network, you'll have to manage IP, so your own OS will be a NAT router.

1. give it an IP address and run a dhcp server

```bash
sudo ip addr add 192.168.123.1/24 dev zos0
md5=$(echo $USER| md5sum )
ULA=${md5:0:2}:${md5:2:4}:${md5:6:4}
sudo ip addr add fd${ULA}::1/64 dev zos0
# you might want to add fe80::1/64
sudo ip addr add fe80::1/64 dev zos0
```

1. configure your firewall to nat this(these) networ(s) and enable forwarding

```bash
sudo iptables -t nat -I POSTROUTING -s 192.168.123.0/24 -j MASQUERADE
sudo ip6tables -t nat -I POSTROUTING -s fd${ULA}::/64 -j MASQUERADE
sudo sysctl -w net.ipv4.ip_forward=1
```

1. and run dnsmasq in a separate terminal, so you can just stop it and have your machine be pristine and clutterless after you're done

```bash
sudo dnsmasq --strict-order \
    --except-interface=lo \
    --interface=zos0 \
    --bind-interfaces \
    --dhcp-range=192.168.123.20,192.168.123.50 \
    --dhcp-range=::1000,::1fff,constructor:zos0,ra-stateless,12h \
    --conf-file="" \
    --pid-file=/var/run/qemu-dnsmasq-zos0.pid \
    --dhcp-leasefile=/var/run/qemu-dnsmasq-zos0.leases \
    --dhcp-no-override
```

1. Now run your vm

```bash
sudo ./vm.sh -n node-01 -c "farmer_id=47 printk.devmsg=on runmode=dev ssh-user=<github username>"
```

where `runmode` is one of `dev` , `test`  or `prod`,
      `farmer_id` is the id of the farm you registered with `tffarmer`
      `ssh-user` is github username provided if a user need to pass ssh-key to the node

NOTE: it is assumed you get a proper IPv6 address, if not, omit the IPv6 parts

## Case2, have the bridge be part of your local lan

1. same, crate your bridge, bring it up

```bash
sudo ip link add zos0 type bridge
sudo sysctl -w net.ipv6.conf.zos0.disable_ipv6=1
sudo ip link set zos0 up
```

where `$yournic` is your wired interface (`ip -br link`)

1. In order to not wreak havoc with your existing network setup (NetworkManager cruft),   it's better to split your nic, so that the rest keeps on working

```bash
sudo ip link add link $yournic name forzos type macvlan mode passthru
sudo sysctl -w net.ipv6.conf.forzos.disable_ipv6=1
sudo ip link set forzos master zos0
sudo ip link set forzos up
```

1. As your bridge is now directly connected to the same network as your machine, htere is nothing more to do; the VM will receive it's IP configuration the same as if it were a physical box connected to your wired lan

1. Now run your vm

```bash
./vm.sh -n node-01 -c "farmer_id=47 version=v3 printk.devmsg=on runmode=dev"
```

where `runmode` is one of `dev` , `test`  or `prod`,
and `farmer_id` is the id of the farm you registered with `tffarmer`

Note: `double quotes around the flags after -c are very important`

## To ssh into the machine

### Authorizing yourself

- To inject you ssh key into zos. You need to add `ssh-user=<github-username>` to your kernel params (with `-c` flag)
- replace `<github-username>` with you actual username. ZOS will use that name to fetch your public ssh key and auto inject it

### SSH to the node

- use `ssh root@{NODE_IP}`

## inspecting the cmdline Arguments

can be done using `cat /proc/cmdline`
