COREX_VERSION="2.1.4"
COREX_CHECKSUM="6427daa42cf93ba65d0377e29ea57148"
COREX_LINK="https://github.com/threefoldtech/corex/releases/download/${COREX_VERSION}/corex-${COREX_VERSION}-amd64-linux-static"


download_corex() {
    download_file ${COREX_LINK} ${COREX_CHECKSUM} corex-${COREX_VERSION}
}

prepare_corex() {
    echo "[+] prepare corex"
    github_name "corex-${COREX_VERSION}"
}

install_corex() {
    echo "[+] install corex"

    mkdir -p "${ROOTDIR}/usr/bin"
    cp -av corex-${COREX_VERSION} "${ROOTDIR}/usr/bin/corex"
    chmod +x "${ROOTDIR}/usr/bin/corex"
}

build_corex() {
    pushd "${DISTDIR}"

    download_corex
    prepare_corex
    install_corex

    popd
}
