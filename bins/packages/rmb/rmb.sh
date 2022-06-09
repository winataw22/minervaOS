RMB_VERSION="0.1.2"
RMB_CHECKSUM="4fefd664f261523b348fc48e9f1c980b"
RMB_LINK="https://github.com/threefoldtech/rmb-rs/releases/download/v${RMB_VERSION}/rmb"

download_rmb() {
    echo "download rmb"
    download_file ${RMB_LINK} ${RMB_CHECKSUM} rmb
}

prepare_rmb() {
    echo "[+] prepare rmb"
    github_name "rmb-${RMB_VERSION}"
}

install_rmb() {
    echo "[+] install rmb"

    mkdir -p "${ROOTDIR}/bin"

    cp ${DISTDIR}/rmb ${ROOTDIR}/bin/
    chmod +x ${ROOTDIR}/bin/*
}

build_rmb() {
    pushd "${DISTDIR}"

    download_rmb
    popd

    prepare_rmb
    install_rmb
}
