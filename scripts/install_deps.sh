#!/bin/bash
set -e

CHV_VERSION="v39.0"
CHV_URL="https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/${CHV_VERSION}/cloud-hypervisor"
RUSTUP_URL="https://sh.rustup.rs"
VIRTIOFSD_REPO="https://gitlab.com/muhamad.azmy/virtiofsd/-/jobs/6547244336/artifacts/download?file_type=archive"
RFS_VERSION="v1.1.1"
RFS_URL="https://github.com/threefoldtech/rfs/releases/download/${RFS_VERSION}/rfs"

install_chv() {
    echo "Installing cloud-hypervisor ${CHV_VERSION} ..."
    wget -q ${CHV_URL} -O /usr/local/bin/cloud-hypervisor
    chmod +x /usr/local/bin/cloud-hypervisor
    setcap cap_sys_admin,cap_dac_override+eip /usr/local/bin/cloud-hypervisor
}


install_virtiofsd() {
    echo "Installing virtiofsd ..."

    # specially needed for virtiofsd bin    
    apt -y update
    apt -y install libseccomp-dev libcap-ng-dev

    curl -L -k -o "/tmp/virtiofsd-rs.zip" ${VIRTIOFSD_REPO}
    unzip -p /tmp/virtiofsd-rs.zip > /usr/local/bin/virtiofsd
    chmod +x /usr/local/bin/virtiofsd
    setcap cap_sys_admin,cap_dac_override+eip /usr/local/bin/virtiofsd
}

install_rfs() {
    echo "Installing rfs ${RFS_VERSION} ..."
    wget -q ${RFS_URL}
    chmod +x rfs
    mv ./rfs /usr/local/bin/rfs1
}

main() {
    # must run as superuser
    if [ $(id -u) != "0" ]; then
    echo "You must be the superuser to run this script" >&2
    exit 1
    fi

    TEMP_DIR=$(mktemp -d)
    pushd $TEMP_DIR

    # install cloud-hypervisor
    if ! command -v cloud-hypervisor &>/dev/null; then
        install_chv
    fi

    # install virtiofsd
    if ! command -v virtiofsd &>/dev/null; then
        
        install_virtiofsd
    fi

    # install rfs
    if ! command -v rfs1 &>/dev/null; then
        install_rfs
    fi

    # install mcopy/mkdosfs needed to create cidata image. comment if you a have the image
    apt -y install dosfstools mtools fuse

    # install screen for managing multiple servers
    # NOTE: rust bins like virtiofsd miss the logs, runs on a screen session to workaround that
    apt -y install screen

    popd
    rm -rf $TEMP_DIR
}

main