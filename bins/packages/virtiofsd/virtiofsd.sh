VIRTIOFSD_VERSION="1.3.0"
VIRTIOFSD_CHECKSUM="0fc813a373eef188dc6b8ca152b2f286"
VIRTIOFSD_LINK="https://gitlab.com/virtio-fs/virtiofsd/uploads/9a4f2261fcb1701f1e709694b5c5d980/virtiofsd-v1.3.0.zip"

download_virtiofsd() {
    download_file ${VIRTIOFSD_LINK} ${VIRTIOFSD_CHECKSUM} virtiofsd-${VIRTIOFSD_VERSION}.zip
}

prepare_virtiofsd() {
    echo "[+] prepare virtiofsd"
    github_name "virtiofsd-${VIRTIOFSD_VERSION}"
}

install_virtiofsd() {
    echo "[+] install virtiofsd"

    mkdir -p "${ROOTDIR}/bin"
    unzip -p virtiofsd-${VIRTIOFSD_VERSION} > "${ROOTDIR}/bin/virtiofsd-rs"
    chmod +x "${ROOTDIR}/bin/virtiofsd-rs"
}

build_virtiofsd() {
    pushd "${DISTDIR}"

    download_virtiofsd
    prepare_virtiofsd
    install_virtiofsd

    popd
}
