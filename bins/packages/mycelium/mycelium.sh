MYCELIUM_VERSION="0.3.0"
MYCELIUM_CHECKSUM="68ac250b23057a18f351a9aa4bc0ef6d"
MYCELIUM_LINK="https://github.com/threefoldtech/mycelium/releases/download/v${MYCELIUM_VERSION}/mycelium-x86_64-unknown-linux-musl.tar.gz"

download_mycelium() {
    download_file ${MYCELIUM_LINK} ${MYCELIUM_CHECKSUM}
}

extract_mycelium() {
    tar -xf ${DISTDIR}/mycelium-x86_64-unknown-linux-musl.tar.gz -C ${WORKDIR}
}

prepare_mycelium() {
    echo "[+] prepare mycelium"
    github_name "mycelium-${MYCELIUM_VERSION}"
}


install_mycelium() {
    echo "[+] install mycelium"

    mkdir -p "${ROOTDIR}/usr/bin"


    cp ${WORKDIR}/mycelium ${ROOTDIR}/usr/bin/mycelium

    chmod +x ${ROOTDIR}/usr/bin/mycelium
}

build_mycelium() {
    pushd "${DISTDIR}"

    download_mycelium
    extract_mycelium

    popd
    pushd ${WORKDIR}

    prepare_mycelium
    install_mycelium

    popd
}
