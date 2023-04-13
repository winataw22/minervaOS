RMB_VERSION="1.0.3"
RMB_CHECKSUM="7b3a62b449ccb0133e98f0c10c0b8c29"
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
