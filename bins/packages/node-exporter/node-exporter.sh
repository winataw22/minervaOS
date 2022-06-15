EXPORTER_VERSION="1.3.1"
EXPORTER_CHECKSUM="9ed75103b9bc65b30e407d4f238f9dea"
EXPORTER_LINK="https://github.com/prometheus/node_exporter/releases/download/v${EXPORTER_VERSION}/node_exporter-${EXPORTER_VERSION}.linux-amd64.tar.gz"

download_exporter() {
    download_file ${EXPORTER_LINK} ${EXPORTER_CHECKSUM} exporter-${EXPORTER_VERSION}.tar.gz
}

prepare_exporter() {
    echo "[+] prepare exporter"
    github_name "node-exporter-${EXPORTER_VERSION}"
}

install_exporter() {
    echo "[+] install exporter"

    mkdir -p "${ROOTDIR}/usr/bin"
    cp -av $1/packages/node-exporter/root/* "${ROOTDIR}/"
    tar -xOf exporter-${EXPORTER_VERSION}.tar.gz node_exporter-${EXPORTER_VERSION}.linux-amd64/node_exporter > "${ROOTDIR}/usr/bin/node_exporter"
    chmod +x "${ROOTDIR}/usr/bin/node_exporter"
}

build_node_exporter() {
    base=$(pwd)
    pushd "${DISTDIR}"

    download_exporter
    prepare_exporter
    install_exporter $base

    popd
}
