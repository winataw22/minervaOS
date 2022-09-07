#!/bin/bash


debug=0
reset=0
image=zos.efi
kernelargs=""
bridge=zos0
graphics="-nographic -nodefaults"
smp=1
mem=3
tpm=0

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
   -m: memory in Gigabytes
   -t: tpm support (requires swtpm)
   -h: help
EOF
   exit 0
}


while getopts "c:n:i:rdtb:gs:m:" opt; do
   case $opt in
   i )  image=$OPTARG ;;
   r )  reset=1 ;;
   d )  debug=1 ;;
   n )  name=$OPTARG ;;
   c )  kernelargs=$OPTARG ;;
   b )  bridge=$OPTARG ;;
   g )  graphics="" ;;
   s )  smp=$OPTARG ;;
   m )  mem=$OPTARG ;;
   t )  tpm=1 ;;
   h )  usage ; exit 0 ;;
   \?)  usage ; exit 1 ;;
   esac
done
shift $(($OPTIND - 1))


cmdline="console=ttyS1,115200n8 zos-debug zos-debug-vm $kernelargs"
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

tpmargs=""
if [[ $tpm -eq "1" ]]; then
   if ! command -v swtpm &> /dev/null; then
      echo "tpm option require `swtpm` please install first"
      exit 1
   fi
   pkill swtpm
   tpm_dir="$vmdir/tpm"
   tpm_socket="$vmdir/swtpm.sock"
   mkdir -p $tpm_dir
   rm $tpm_socket &> /dev/null || true
   # runs in the backgroun
   swtpm \
      socket --tpm2 \
      --tpmstate dir=$tpm_dir \
      --ctrl type=unixio,path=$vmdir/swtpm.sock \
      --log level=20 &> tpm.logs &

   while [ ! -S "$tpm_socket" ]; do
      echo "waiting for tpm"
      sleep 1s
   done
   sleep 1s
   tpmargs="-chardev socket,id=chrtpm,path=${tpm_socket} -tpmdev emulator,id=tpm0,chardev=chrtpm -device tpm-tis,tpmdev=tpm0"
fi

echo "boot $image"

qemu-system-x86_64 -kernel $image \
    -m $(( mem * 1024 )) \
    -enable-kvm \
    -cpu host,host-phys-bits \
    -smp $smp \
    -uuid $uuid \
    -netdev bridge,id=zos0,br=${bridge} -device virtio-net-pci,netdev=zos0,mac="${basemac}1" \
    -drive file=fat:rw:$basepath/overlay,format=raw \
    -append "${cmdline}" \
    -drive file=$vmdir/vda.qcow2,if=virtio -drive file=$vmdir/vdb.qcow2,if=virtio \
    -drive file=$vmdir/vdc.qcow2,if=virtio -drive file=$vmdir/vdd.qcow2,if=virtio \
    -drive file=$vmdir/vde.qcow2,if=virtio \
    -serial null -serial mon:stdio \
    ${graphics} \
    ${tpmargs} \
    ;
