CLOUDHYPERVISOR_VERSION="19.0"
CLOUDHYPERVISOR_CHECKSUM="fd91ecc9c9bb7599beb39bc3a8206389"
CLOUDHYPERVISOR_LINK="https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v${CLOUDHYPERVISOR_VERSION}/cloud-hypervisor-static"


download_cloudhypervisor() {
    echo "down"
    download_file ${CLOUDHYPERVISOR_LINK} ${CLOUDHYPERVISOR_CHECKSUM} cloud-hypervisor-${CLOUDHYPERVISOR_VERSION}
}


prepare_cloudhypervisor() {
    echo "[+] prepare cloud-hypervisor"
    github_name "cloud-hypervisor-${CLOUDHYPERVISOR_VERSION}"
}

install_cloudhypervisor() {
    echo "[+] install cloud-hypervisor"

    mkdir -p "${ROOTDIR}/usr/bin"

    cp ${DISTDIR}/cloud-hypervisor-${CLOUDHYPERVISOR_VERSION} ${ROOTDIR}/usr/bin/cloud-hypervisor
    chmod +x ${ROOTDIR}/usr/bin/*
}

build_cloudhypervisor() {
    pushd "${DISTDIR}"

    download_cloudhypervisor
    prepare_cloudhypervisor
    install_cloudhypervisor

    popd
}
