TAILSTREAM_VERSION="0.1.6"
TAILSTREAM_CHECKSUM="adac25b1a6bbdd6a207156167bd323aa"
TAILSTREAM_LINK="https://github.com/threefoldtech/tailstream/releases/download/v${TAILSTREAM_VERSION}/tailstream"

download_tailstream() {
    download_file ${TAILSTREAM_LINK} ${TAILSTREAM_CHECKSUM} tailstream-${TAILSTREAM_VERSION}
}

prepare_tailstream() {
    echo "[+] prepare tailstream"
    github_name "tailstream-${TAILSTREAM_VERSION}"
}

install_tailstream() {
    echo "[+] install tailstream"

    mkdir -p "${ROOTDIR}/bin"
    cp -av tailstream-${TAILSTREAM_VERSION} "${ROOTDIR}/bin/tailstream"
    chmod +x "${ROOTDIR}/bin/tailstream"
}

build_tailstream() {
    pushd "${DISTDIR}"

    download_tailstream
    prepare_tailstream
    install_tailstream

    popd
}
