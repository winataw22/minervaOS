CLOUDCONSOLE_VERSION="v0.1.2"
CLOUDCONSOLE_CHECKSUM="ae22f069ec74d74b25ebbbddbc390fa2"
CLOUDCONSOLE_LINK="https://github.com/threefoldtech/cloud-console/releases/download/${CLOUDCONSOLE_VERSION}/cloud-console"

download_cloudconsole() {
    echo "downloading cloud-console"
    download_file ${CLOUDCONSOLE_LINK} ${CLOUDCONSOLE_CHECKSUM} cloud-console-${CLOUDCONSOLE_VERSION}
}


prepare_cloudconsole() {
    echo "[+] prepare cloud-console"
    github_name "cloud-console-${CLOUDCONSOLE_VERSION}"
}

install_cloudconsole() {
    echo "[+] install cloud-console"

    mkdir -p "${ROOTDIR}/usr/bin"

    cp ${DISTDIR}/cloud-console-${CLOUDCONSOLE_VERSION} ${ROOTDIR}/usr/bin/cloud-console
    chmod +x ${ROOTDIR}/usr/bin/*
}

build_cloudconsole() {
    pushd "${DISTDIR}"

    download_cloudconsole
    prepare_cloudconsole
    install_cloudconsole

    popd
}
