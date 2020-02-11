#!/bin/bash


debug=0
reset=0
image=zos.efi
kernelargs=""
bridge=zos0
graphics="-nographic -nodefaults"
smp=1

usage() {
   cat <<EOF
Usage: vm -n $name [ -r ] [ -d ]
   -n $name: name of the vm
   -d: run in debug mode
   -r: reset vm (recreate disks)
   -c: Kernel command
   -s: number of virtual cpus
   -i: kernel image to boot
   -b: bridge for network (default zos)
   -g: open GUI
   -h: help

EOF
   exit 0
}


while getopts "c:n:i:rdb:gs:" opt; do
   case $opt in
   i )  image=$OPTARG ;;
   r )  reset=1 ;;
   d )  debug=1 ;;
   n )  name=$OPTARG ;;
   c )  kernelargs=$OPTARG ;;
   b )  bridge=$OPTARG ;;
   g )  graphics="" ;;
   s )  smp=$OPTARG ;;
   h )  usage ; exit 0 ;;
   \?)  usage ; exit 1 ;;
   esac
done
shift $(($OPTIND - 1))


cmdline="console=ttyS1,115200n8 $kernelargs"
basepath=$(dirname $0)
vmdir="$basepath/$name"
md5=$(echo -n $name | md5sum | awk '{print $1}')
uuid="${md5:0:8}-${md5:8:4}-${md5:12:4}-${md5:16:4}-${md5:20}"
basemac="54:${md5:2:2}:${md5:4:2}:${md5:6:2}:${md5:8:2}:${md5:10:1}"

createdisks() {
    qemu-img create -f qcow2 $vmdir/vda.qcow2 500G
    qemu-img create -f qcow2 $vmdir/vdb.qcow2 500G
    qemu-img create -f qcow2 $vmdir/vdc.qcow2 500G
    qemu-img create -f qcow2 $vmdir/vdd.qcow2 500G
    qemu-img create -f qcow2 $vmdir/vde.qcow2 500G

}

if [[ ! -d "$vmdir" || "$reset" -eq 1 ]]; then
    mkdir -p "$vmdir"
    createdisks
fi

if ps -eaf | grep -v grep | grep "$uuid" > /dev/null; then
    echo "VM $name is already running"
    exit 1
fi

echo "boot $image"

qemu-system-x86_64 -kernel $image \
    -m 3072 -enable-kvm -cpu host -smp $smp \
    -uuid $uuid \
    -netdev bridge,id=zos0,br=${bridge} -device virtio-net-pci,netdev=zos0,mac="${basemac}1" \
    -drive file=fat:rw:$basepath/overlay,format=raw \
    -append "${cmdline}" \
    -drive file=$vmdir/vda.qcow2,if=virtio -drive file=$vmdir/vdb.qcow2,if=virtio \
    -drive file=$vmdir/vdc.qcow2,if=virtio -drive file=$vmdir/vdd.qcow2,if=virtio \
    -drive file=$vmdir/vde.qcow2,if=virtio \
    -serial null -serial mon:stdio \
    ${graphics}
