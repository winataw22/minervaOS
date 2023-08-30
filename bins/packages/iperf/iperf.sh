IPERF_VERSION="3.14"
IPERF_CHECKSUM="9ef3769b601d79250bb1593b0146dbcb"
IPERF_LINK="https://github.com/userdocs/iperf3-static/releases/download/${IPERF_VERSION}/iperf3-amd64"

download_iperf() {
    echo "downloading iperf"
    download_file ${IPERF_LINK} ${IPERF_CHECKSUM} iperf-${IPERF_VERSION}
}


prepare_iperf() {
    echo "[+] prepare iperf"
    github_name "iperf-${IPERF_VERSION}"
}

install_iperf() {
    echo "[+] install iperf"

    mkdir -p "${ROOTDIR}/usr/bin"

    cp ${DISTDIR}/iperf-${IPERF_VERSION} ${ROOTDIR}/usr/bin/iperf
    chmod +x ${ROOTDIR}/usr/bin/*
}

build_iperf() {
    pushd "${DISTDIR}"

    download_iperf
    prepare_iperf
    install_iperf

    popd
}
