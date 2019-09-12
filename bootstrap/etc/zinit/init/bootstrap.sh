set -x

DEFAULT_FLIST=azmy/zos-refs_heads_master.flist

# bootflist reads the boot flist name from kernel cmd
# TODO: this should probably be not allowed at somepoint to
function bootflist() {
    for param in $(strings /proc/cmdline); do
        if [[ "${param:0:6}" == "flist=" ]]
        then
            echo ${param#flist=}
            return 0
        fi
    done

    echo ${DEFAULT_FLIST}
}

BOOTFLIST=https://hub.grid.tf/$(bootflist)

echo "Bootstraping: ${BOOTFLIST}"

# helper retry function
# the retry function never give up because the
# bootstrap must succeed. otherwise the node
# will not be functional. So no reason to give
# up
function retry() {
    until $@; do
        sleep 1s
        echo "retrying: $@"
    done
}

BS=/tmp/bootstrap
mkdir -p ${BS}

## Prepare a tmpfs for 0-fs cache
mount -t tmpfs -o size=512M tmpfs ${BS}

cd ${BS}
mkdir -p root
retry wget -O machine.flist ${BOOTFLIST}

g8ufs --backend ${BS}/backend --meta machine.flist root &

retry mountpoint root

## move to root
cd root
cp -a * /
### filesystem is ready

for file in $(ls etc/zinit/*.yaml); do
    file=$(basename ${file})
    name="${file%.*}"
    zinit monitor ${name}
done

cd /tmp
umount -fl ${BS}/root
umount -fl ${BS}
