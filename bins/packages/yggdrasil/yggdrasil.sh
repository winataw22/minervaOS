YGGDRASIL_VERSION="0.4.2"
YGGDRASIL_CHECKSUM="bac68c7b5de45c37add50138df8f2c7f"
YGGDRASIL_LINK="https://github.com/yggdrasil-network/yggdrasil-go/releases/download/v${YGGDRASIL_VERSION}/yggdrasil-${YGGDRASIL_VERSION}-amd64.deb"

download_yggdrasil() {
    download_file ${YGGDRASIL_LINK} ${YGGDRASIL_CHECKSUM} yggdrasil-v${YGGDRASIL_VERSION}.deb
}

extract_yggdrasil() {
    apt-get install ./yggdrasil-v${YGGDRASIL_VERSION}.deb
}

prepare_yggdrasil() {
    echo "[+] prepare yggdrasil"
    github_name "yggdrasil-${YGGDRASIL_VERSION}"
}

install_yggdrasil() {
    echo "[+] install yggdrasil"

    mkdir -p "${ROOTDIR}/usr/bin"
    mkdir -p "${ROOTDIR}/etc/yggdrasil"
    mkdir -p "${ROOTDIR}/etc/zinit"

    cp -av $(which yggdrasil) "${ROOTDIR}/usr/bin/"
    cp -av $(which yggdrasilctl) "${ROOTDIR}/usr/bin/"
}

build_yggdrasil() {
    pushd "${DISTDIR}"
    download_yggdrasil
    extract_yggdrasil
    popd

    prepare_yggdrasil
    install_yggdrasil
}
