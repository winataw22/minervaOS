set -e

flistfile=/tmp/flist.name
infofile=/tmp/flist.info

# helper retry function
# the retry function never give up because the
# bootstrap must succeed. otherwise the node
# will not be functional. So no reason to give
# up
retry() {
    until $@; do
        sleep 1s
        echo "retrying: $@"
    done
}

param() {
    local key="$1="
    for arg in $(strings /proc/cmdline); do
        if [[ "${arg:0:${#key}}" == "${key}" ]]
        then
            echo ${arg#${key}}
            return 0
        fi
    done

    return 1
}

default_param() {
    local key=$1
    local default=$2

    if ! param "${key}"; then
        echo ${default}
    fi
}

runmode=$(default_param runmode prod)

# set default production flist
flist=tf-zos/zos:production:latest.flist

case "${runmode}" in
    prod)
    ;;
    dev)
        flist=tf-zos/zos:development:latest.flist
    ;;
    test)
        flist=tf-zos/zos:testing:latest.flist
    ;;
    *)
        echo "Invalid run mode '${runmode}'. fall back to production"
    ;;
esac

# track which flist used for booting
echo ${flist} > ${flistfile}
chmod 0400 ${flistfile}

bootflist=https://hub.grid.tf/${flist}
bootflistinfo=https://hub.grid.tf/api/flist/${flist}/light

echo "Bootstraping with: ${bootflist}"
retry wget -O ${infofile} ${bootflistinfo}
chmod 0400 ${infofile}

bs=/tmp/bootstrap
mkdir -p ${bs}

## Prepare a tmpfs for 0-fs cache
mount -t tmpfs -o size=512M tmpfs ${bs}

cd ${bs}
mkdir -p root
retry wget -O machine.flist ${bootflist}

g8ufs --backend ${bs}/backend --meta machine.flist root &

retry mountpoint root

echo "Installing core services"
## move to root
cd root
cp -a * /
### filesystem now has all the binaries

for file in $(ls etc/zinit/*.yaml); do
    file=$(basename ${file})
    name="${file%.*}"
    if ! zinit monitor ${name}; then
        zinit kill ${name} || true
    fi
done

echo "Installation complete"

cd /tmp
umount -fl ${bs}/root
umount -fl ${bs}
