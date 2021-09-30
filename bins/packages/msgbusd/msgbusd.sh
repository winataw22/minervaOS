MSGBUSD_VERSION="0.1.2"
MSGBUSD_CHECKSUM="bd4ddd29658147efeed4d5cbac8bfdd9"
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
