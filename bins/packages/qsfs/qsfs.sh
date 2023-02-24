QSFS_VERSION="0.2.0-rc2"
QSFS_IMAGE="ghcr.io/threefoldtech/qsfs"

download_qsfs() {
    echo "download qsfs"
    download_docker_image ${QSFS_IMAGE}:${QSFS_VERSION} qsfs-${QSFS_VERSION}.tar
}

prepare_qsfs() {
    echo "[+] prepare qsfs"
    github_name "qsfs-${QSFS_VERSION}"
}

install_qsfs() {
    echo "[+] install qsfs"

    mkdir -p "${ROOTDIR}"

    extract_docker_image "${DISTDIR}/qsfs-${QSFS_VERSION}.tar" "${ROOTDIR}"
}

build_qsfs() {
    pushd "${DISTDIR}"

    download_qsfs
    popd

    prepare_qsfs
    install_qsfs
}
