RFS_VERSION="2.0.0"
RFS_CHECKSUM="d6e1c3036d8b0d7d6df4d67fe17a0502"
RFS_LINK="https://github.com/threefoldtech/rfs/releases/download/v${RFS_VERSION}/rfs"

download_rfs() {
    download_file ${RFS_LINK} ${RFS_CHECKSUM} rfs-${RFS_VERSION}
}

prepare_rfs() {
    echo "[+] prepare rfs"
    github_name "rfs-${RFS_VERSION}"
}

install_rfs() {
    echo "[+] install rfs"

    mkdir -p "${ROOTDIR}/sbin"
    cp -av rfs-${RFS_VERSION} "${ROOTDIR}/sbin/rfs"
    chmod +x "${ROOTDIR}/sbin/rfs"
}

build_rfs() {
    pushd "${DISTDIR}"

    download_rfs
    prepare_rfs
    install_rfs

    popd
}
