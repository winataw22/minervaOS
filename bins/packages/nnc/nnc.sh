NNC_VERSION="1.0.0-rc2"
NNC_CHECKSUM="7fbadaae30c19b79e6348bf1bd1d29fc"
NNC_LINK="https://github.com/threefoldtech/nnc/releases/download/v${NNC_VERSION}/nnc"

download_nnc() {
    echo "download nnc"
    download_file ${NNC_LINK} ${NNC_CHECKSUM} nnc-peer
}

prepare_nnc() {
    echo "[+] prepare nnc"
    github_name "nnc-${NNC_VERSION}"
}

install_nnc() {
    echo "[+] install nnc"

    mkdir -p "${ROOTDIR}/bin"

    cp ${DISTDIR}/nnc-peer ${ROOTDIR}/bin/nnc
    chmod +x ${ROOTDIR}/bin/*
    strip ${ROOTDIR}/bin/*
}

build_nnc() {
    pushd "${DISTDIR}"

    download_nnc
    popd

    prepare_nnc
    install_nnc
}
