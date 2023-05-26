ZUFS_VERSION="1.1.1"
ZUFS_CHECKSUM="974b8dc45ae9c1b00238a79b0f4fc9de"
ZUFS_LINK="https://github.com/threefoldtech/rfs/releases/download/v${ZUFS_VERSION}/rfs"

download_zufs() {
    download_file ${ZUFS_LINK} ${ZUFS_CHECKSUM} rfs-${ZUFS_VERSION}
}

prepare_zufs() {
    echo "[+] prepare 0-fs"
    github_name "0-fs-${ZUFS_VERSION}"
}

install_zufs() {
    echo "[+] install 0-fs"

    mkdir -p "${ROOTDIR}/sbin"
    cp -av rfs-${ZUFS_VERSION} "${ROOTDIR}/sbin/g8ufs"
    chmod +x "${ROOTDIR}/sbin/g8ufs"
}

build_zufs() {
    pushd "${DISTDIR}"

    download_zufs
    prepare_zufs
    install_zufs

    popd
}
