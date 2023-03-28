MISC_VERSION="1"

prepare_misc() {
    echo "[+] prepare 0-fs"
    github_name "misc-${MISC_VERSION}"
}

install_misc() {
    echo "[+] install misc"

    mkdir -p "${ROOTDIR}/"
    cp -av $1/packages/misc/root/* "${ROOTDIR}/"
}

build_misc() {
    base=$(pwd)
    pushd "${DISTDIR}"

    prepare_misc
    install_misc $base

    popd
}
