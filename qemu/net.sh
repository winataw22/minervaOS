#!/bin/bash

# This is the same as the first case at qemu/README.md in a single script

sudo ip link add zos0 type bridge
sudo ip link set zos0 up

sudo ip addr add 192.168.123.1/24 dev zos0
md5=$(echo $USER| md5sum )
ULA=${md5:0:2}:${md5:2:4}:${md5:6:4}
sudo ip addr add fd${ULA}::1/64 dev zos0
# you might want to add fe80::1/64
sudo ip addr add fe80::1/64 dev zos0

sudo iptables -t nat -I POSTROUTING -s 192.168.123.0/24 -j MASQUERADE
sudo ip6tables -t nat -I POSTROUTING -s fd${ULA}::/64 -j MASQUERADE
sudo iptables -t filter -I FORWARD --source 192.168.123.0/24 -j ACCEPT
sudo iptables -t filter -I FORWARD --destination 192.168.123.0/24 -j ACCEPT
sudo sysctl -w net.ipv4.ip_forward=1

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
