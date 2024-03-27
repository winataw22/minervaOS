MDADM_VERSION="4.3"
MDADM_LINK="https://kernel.googlesource.com/pub/scm/utils/mdadm/mdadm"

download_mdadm() {
    download_git $MDADM_LINK "mdadm-$MDADM_VERSION"
}

prepare_mdadm() {
    echo "[+] prepare mdadm"
    github_name "mdadm-${MDADM_VERSION}"
}

compile_mdadm() {
    echo "[+] compiling mdadm"
    make
}

install_mdadm() {
    echo "[+] installing mdadm"
    mkdir -p "${ROOTDIR}/bin"
    cp mdadm "${ROOTDIR}/bin/mdadm"
}

build_mdadm() {
    apt-get install -y \
        build-essential \
        git \
        libudev-dev

    pushd "${WORKDIR}"

    download_mdadm
    prepare_mdadm

    pushd "mdadm"
    compile_mdadm
    install_mdadm
    popd

    popd

}
