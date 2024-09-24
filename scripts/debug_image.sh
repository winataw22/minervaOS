#!/bin/bash

# Constants
readonly SOCKET="/tmp/virtiofs.sock"
readonly CH_API_SOCKET="/tmp/ch-api.sock"
readonly OVERLAYFS="/tmp/overlay"
readonly CCFLIST="https://hub.grid.tf/tf-autobuilder/cloud-container-9dba60e.flist"
readonly MACHINE_TYPE="machine"
readonly CONTAINER_TYPE="container"

# Defaults
cidata="/tmp/cidata.img"
cmdline="rw console=ttyS0 reboot=k panic=1 root=vroot rootfstype=virtiofs rootdelay=30"

# Globals
declare -A options=(
    [--debug]="false"
    [--cidata]=""
    [--kernel]=""
    [--initramfs]=""
    [--init]=""
    [--user]="user"
    [--pass]="pass"
    [--name]="cloud"
)

image_type=$MACHINE_TYPE
install_deps=false
init=""
kernel=""
initramfs=""
pids=()

# Functions
fail() {
    echo "Error: $*" >&2
    exit 1
}

usage() {
    echo "Usage: debug_image.sh --image IMAGE [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --image IMAGE         Path to the directory containing the rootfs or the flist url."
    echo "  --kernel KERNEL       Path to kernel file (compressed or uncompressed). Default: '<rootfs>/boot/vmlinuz'."
    echo "  --initramfs INITRAMFS Path to initramfs image. Default: '<rootfs>/boot/initrd.img'."
    echo "  --init INIT           Entrypoint for the machine."
    echo "  --cidata CIDATA       Path to optional cloud init image."
    echo "  --user USER           Cloud-init username. Default: 'user'."
    echo "  --pass PASS           Cloud-init password. Default: 'pass'."
    echo "  --name NAME           Cloud-init machine name. Default: 'cloud'."
    echo ""
    echo "  -d                    Enable debugging mode with 'set -x'."
    echo "  -h                    Show this help message."
    echo "  -c                    Run on container mode (will provide a kernel and initrd)."
    echo "  -i                    Don't fail on missing deps and install them."
    echo ""
    echo "Example:"
    echo "  debug_image.sh --image /path/to/rootfs --debug true"
}

handle_options() { 
    while [[ "$#" -gt 0 ]]; do
        case $1 in
            --image|--cidata|--kernel|--initramfs|--init|--user|--pass|--name)
                options["$1"]="$2"
                shift 2
                ;;
            -d)
                set -x
                shift
                ;;
            -h)
                usage
                exit 0
                ;;
            -c)
                image_type=$CONTAINER_TYPE
                shift
                ;;
            -i)
                install_deps=true
                shift
                ;;
            *)
                usage
                exit 1
                ;;
        esac
    done

    if [[ -z "${options[--image]}" ]]; then
        fail "rootfs image is required"
    fi
}

check_or_install_deps() {
    if [ "$install_deps" = true ]; then
        bash ./install_deps.sh
        return
    fi

    for cmd in cloud-hypervisor virtiofsd rfs1 screen; do
        if ! command -v $cmd &>/dev/null; then
            fail "$cmd command not found"
        fi
    done

    if [[ -n "${options[--cidata]}" ]]; then
        for cmd in mcopy mkdosfs; do
            if ! command -v $cmd &>/dev/null; then
                fail "$cmd command not found"
            fi
        done
    fi
}

decide_kernel() {
    local init="${options[--init]-}"
    if [[ -n "$init" ]]; then
        cmdline+=" init=$init"
    fi

    # if options provided, use them and return
    kernel="${options[--kernel]}"
    initramfs="${options[--initramfs]}"
    if [[ -n "${kernel}" && -n "${initramfs}" ]]; then
        return
    fi

    # if container mode, use cloud-container kernel
    if [[ ${image_type} == $CONTAINER_TYPE ]]; then
        kernel="$OVERLAYFS/kernel"
        initramfs="$OVERLAYFS/initramfs-linux.img"
        return
    fi

    # default is to run a full vm
    kernel="$OVERLAYFS/boot/vmlinuz"
    initramfs="$OVERLAYFS/boot/initrd.img"
}

create_cidata() {
    if [[ -f "${options[--cidata]}" ]]; then
        cidata="${options[--cidata]}"
        return
    fi

    local user="${options[--user]}"
    local pass="${options[--pass]}"
    local name="${options[--name]}"

    echo '#cloud-config' > user-data
    echo 'users:' >> user-data
    echo "  - name: $user" >> user-data
    echo "    plain_text_passwd: $pass" >> user-data
    echo "    shell: /bin/bash" >> user-data
    echo "    lock_passwd: false" >> user-data
    echo "    sudo: ALL=(ALL) NOPASSWD:ALL" >> user-data

    echo "instance-id: $name" > meta-data
    echo "local-hostname: $name" >> meta-data

    rm -f "${cidata}"

    label="CIDATA"
    if [[ "${image_type}" == $CONTAINER_TYPE ]]; then
        label="cidata"
    fi

    mkdosfs -n "${label}" -C "${cidata}" 8192
    mcopy -oi "${cidata}" -s user-data ::
    mcopy -oi "${cidata}" -s meta-data ::

    rm -f user-data meta-data
}

mount_flist() {
    flist_url=$1
    mountpoint=$2

    echo "mounting $flist_url at $mountpoint"

    flist_path=$(basename "$flist_url")
    flist_path="/tmp/$flist_path"

    if [ ! -f "$flist_path" ]; then
        wget $flist_url -O $flist_path
    fi

    mkdir -p "$mountpoint"

    rfs1 --meta "$flist_path" "$mountpoint" &
    pids+=($!)

    while [ -z "$(ls -A "$mountpoint")" ]; do
        echo "waiting for flist mount"
        sleep 1
    done
}

prepare_rootfs() {
    local image="${options[--image]}"

    lowerdir="$image"
    if [[ ${image} == *.flist ]]; then
        path="/tmp/flist"
        mount_flist "$image" "$path"

        lowerdir="$path"
    fi

    if [[ ${image_type} == $CONTAINER_TYPE ]]; then
        path="/tmp/cloud-container"
        mount_flist "$CCFLIST" "$path"

        lowerdir="$lowerdir:$path"
    fi

    echo "Mounting overlay"
    mkdir -p /tmp/upper /tmp/workdir "$OVERLAYFS"
    mount \
        -t overlay \
        -o lowerdir="$lowerdir",upperdir=/tmp/upper,workdir=/tmp/workdir \
        none \
        "$OVERLAYFS"

    echo "Starting virtiofs"
    # a trick to not mess the logs, it asks for sudo
    screen -dmS virtiofsd_session sudo virtiofsd --socket-path="$SOCKET" --shared-dir="$OVERLAYFS" --cache=never
    pids+=($!)
}

cleanup() {
    screen -S virtiofsd_session -X kill &>/dev/null || true

    dirs=("$OVERLAYFS" /tmp/flist /tmp/cloud-container /tmp/upper /tmp/workdir)
    for dir in "${dirs_to_umount[@]}"; do
        umount "$dir" &>/dev/null || true
        rm -rf "$dir" &>/dev/null || true
    done

    for pid in "${pids[@]}"; do
        kill "$pid" &>/dev/null || true
    done

    echo "CLEAR!"
}

boot() {
    cloud-hypervisor \
        --cpus boot=1,max=1 \
        --memory size=1024M,shared=on \
        --kernel "${kernel}" \
        --initramfs "${initramfs}" \
        --fs tag=vroot,socket="$SOCKET" \
        --disk path="$cidata" \
        --cmdline "$cmdline" \
        --api-socket="$CH_API_SOCKET" \
        --serial tty \
        --console off
}

# must run as superuser
if [ $(id -u) != "0" ]; then
echo "You must be the superuser to run this script" >&2
exit 1
fi

cleanup

echo "Starting ..."
handle_options "$@"

echo "Check or install deps"
check_or_install_deps

echo "Decide kernel"
decide_kernel

echo "Creating cidata"
create_cidata

echo "Prepare rootfs"
prepare_rootfs

trap cleanup EXIT
trap cleanup ERR

echo "Booting ..."
boot
