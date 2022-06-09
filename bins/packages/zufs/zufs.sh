ZUFS_VERSION="0.2.6"
ZUFS_CHECKSUM="8d576939d75b12613abdfc1bd18204f6"
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
