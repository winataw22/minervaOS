ZUFS_VERSION="development"
ZUFS_CHECKSUM="d28ec96dd7586f7a1763c54c5448921e"
ZUFS_REPOSITORY="https://github.com/threefoldtech/0-fs"

dependencies_zufs() {
    apt-get install -y git btrfs-tools libseccomp-dev build-essential pkg-config

    . ${PKGDIR}/../golang/golang.sh
    build_golang

    TF_HOME="${GOPATH}/src/github.com/threefoldtech"
}

download_zufs() {
    download_git ${ZUFS_REPOSITORY} ${ZUFS_VERSION}
}

extract_zufs() {
    event "refreshing" "0-fs-${ZUFS_VERSION}"
    mkdir -p ${TF_HOME}
    rm -rf ${TF_HOME}/0-fs
    cp -a ${DISTDIR}/0-fs ${TF_HOME}/
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
    pushd ${TF_HOME}/0-fs

    prepare_zufs
    compile_zufs
    install_zufs

    popd
}

