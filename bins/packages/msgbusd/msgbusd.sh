MSGBUSD_VERSION="0.1.1"
MSGBUSD_CHECKSUM="9cbd3290798b3b56c5c9f3f5889fa8e6"
MSGBUSD_LINK="https://github.com/threefoldtech/rmb-go/releases/download/v${MSGBUSD_VERSION}/msgbusd"

download_msgbusd() {
    echo "download msgbusd"
    download_file ${MSGBUSD_LINK} ${MSGBUSD_CHECKSUM} msgbusd
}

prepare_msgbusd() {
    echo "[+] prepare msgbusd"
    github_name "msgbusd-${MSGBUSD_VERSION}"
}

install_msgbusd() {
    echo "[+] install msgbusd"

    mkdir -p "${ROOTDIR}/bin"

    cp ${DISTDIR}/msgbusd ${ROOTDIR}/bin/
    chmod +x ${ROOTDIR}/bin/*
}

build_msgbusd() {
    pushd "${DISTDIR}"

    download_msgbusd
    popd

    prepare_msgbusd
    install_msgbusd

}
