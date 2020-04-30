SHIMLOGS_VERSION="0.1"
SHIMLOGS_CHECKSUM="f97c7067064d0cd1643c8141d982c293"
SHIMLOGS_LINK="https://github.com/threefoldtech/shim-logs/archive/v${SHIMLOGS_VERSION}.tar.gz"

dependencies_shimlogs() {
    apt-get install -y libjansson-dev libhiredis-dev build-essential
}

download_shimlogs() {
    download_file ${SHIMLOGS_LINK} ${SHIMLOGS_CHECKSUM} shim-logs-${SHIMLOGS_VERSION}.tar.gz
}

extract_shimlogs() {
    tar -xf ${DISTDIR}/shim-logs-${SHIMLOGS_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_shimlogs() {
    echo "[+] prepare shim-logs"
    github_name "shim-logs-${SHIMLOGS_VERSION}"
}

compile_shimlogs() {
    echo "[+] compile shim-logs"
    make
}

install_shimlogs() {
    echo "[+] install shim-logs"

    mkdir -p "${ROOTDIR}/bin"

    cp shim-logs ${ROOTDIR}/bin/shim-logs
    chmod +x ${ROOTDIR}/bin/shim-logs
}

build_shimlogs() {
    pushd "${DISTDIR}"

    dependencies_shimlogs
    download_shimlogs
    extract_shimlogs

    popd
    pushd ${WORKDIR}/shim-logs-${SHIMLOGS_VERSION}

    prepare_shimlogs
    compile_shimlogs
    install_shimlogs

    popd
}

