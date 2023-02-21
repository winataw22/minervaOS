RMB_VERSION="1.0.1-rc6"
RMB_CHECKSUM="117532f5da654d3a1cbe42e2edf9702e"
RMB_LINK="https://github.com/threefoldtech/rmb-rs/releases/download/v${RMB_VERSION}/rmb-peer"

download_rmb() {
    echo "download rmb"
    download_file ${RMB_LINK} ${RMB_CHECKSUM} rmb-peer
}

prepare_rmb() {
    echo "[+] prepare rmb"
    github_name "rmb-${RMB_VERSION}"
}

install_rmb() {
    echo "[+] install rmb"

    mkdir -p "${ROOTDIR}/bin"

    cp ${DISTDIR}/rmb-peer ${ROOTDIR}/bin/rmb
    chmod +x ${ROOTDIR}/bin/*
    strip ${ROOTDIR}/bin/*
}

build_rmb() {
    pushd "${DISTDIR}"

    download_rmb
    popd

    prepare_rmb
    install_rmb
}
