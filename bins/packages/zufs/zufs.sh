ZUFS_VERSION="0.2.7"
ZUFS_CHECKSUM="5a08c6a12aefbd139d39f1ef9194ad5b"
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
