ZUFS_VERSION="2.0.4"
ZUFS_CHECKSUM="6f42aaee6449556ab7d5ea9567c54347"
ZUFS_LINK="https://github.com/threefoldtech/0-fs/archive/v${ZUFS_VERSION}.tar.gz"

dependencies_zufs() {
    apt-get install -y git btrfs-tools libseccomp-dev build-essential pkg-config

    . ${PKGDIR}/../golang/golang.sh
    build_golang
}

download_zufs() {
    download_file ${ZUFS_LINK} ${ZUFS_CHECKSUM} 0-fs-${ZUFS_VERSION}.tar.gz
}

extract_zufs() {
    tar -xf ${DISTDIR}/0-fs-${ZUFS_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_zufs() {
    echo "[+] prepare 0-fs"
    github_name "0-fs-${ZUFS_VERSION}"
}

compile_zufs() {
    echo "[+] compiling 0-fs"
    make
}

install_zufs() {
    echo "[+] install 0-fs"

    mkdir -p "${ROOTDIR}/sbin"
    cp -av g8ufs "${ROOTDIR}/sbin/"
}

build_zufs() {
    pushd "${DISTDIR}"

    dependencies_zufs
    download_zufs
    extract_zufs

    popd
    pushd ${WORKDIR}/0-fs-${ZUFS_VERSION}

    prepare_zufs
    compile_zufs
    install_zufs

    popd
}
