VIRTIOFSD_VERSION="v1.10.2"
VIRTIOFSD_CHECKSUM="df1ed186ee84843605137758e2aa6e80"
VIRTIOFSD_LINK="https://gitlab.com/muhamad.azmy/virtiofsd/-/jobs/6547244336/artifacts/download?file_type=archive"

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
