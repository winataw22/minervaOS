CLOUDHYPERVISOR_VERSION="39.0"
CLOUDHYPERVISOR_CHECKSUM="c6f0f32b8ed6e68e0f9ddff805d912c7"
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
