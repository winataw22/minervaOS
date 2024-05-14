#!/bin/bash

mkdir $1
sudo debootstrap jammy $1  http://archive.ubuntu.com/ubuntu

sudo arch-chroot $1 /bin/bash <<EOF

export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
rm /etc/resolv.conf
echo 'nameserver 1.1.1.1' > /etc/resolv.conf

apt-get update -y
apt-get install -y cloud-init openssh-server curl
cloud-init clean

apt-get install -y linux-modules-extra-5.15.0-25-generic
echo 'fs-virtiofs' >> /etc/initramfs-tools/modules
update-initramfs -c -k all

EOF