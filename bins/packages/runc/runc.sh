RUNC_VERSION="1.0.0-rc9"
RUNC_CHECKSUM="e88bcb1a33e7ff0bfea495f7263826c2"
RUNC_LINK="https://github.com/opencontainers/runc/archive/v${RUNC_VERSION}.tar.gz"

dependencies_runc() {
    apt-get install -y btrfs-tools libseccomp-dev build-essential pkg-config

    if [ -z $GOPATH ]; then
        if command -v go > /dev/null; then
            export GOPATH=$(go env GOPATH)
        else
            curl -L https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz > /tmp/go1.13.1.linux-amd64.tar.gz
            tar -C /usr/local -xzf /tmp/go1.13.1.linux-amd64.tar.gz
            mkdir -p /gopath

            export PATH=$PATH:/usr/local/go/bin
            export GOPATH=/gopath
        fi
    fi

    RUNC_HOME="${GOPATH}/src/github.com/opencontainers"
}

download_runc() {
    download_file ${RUNC_LINK} ${RUNC_CHECKSUM} runc-${RUNC_VERSION}.tar.gz
}

extract_runc() {
    mkdir -p ${RUNC_HOME}
    rm -rf ${RUNC_HOME}/runc

    pushd ${RUNC_HOME}

    echo "[+] extracting: runc-${RUNC_VERSION}"
    tar -xf ${DISTDIR}/runc-${RUNC_VERSION}.tar.gz -C .
    mv runc-${RUNC_VERSION} runc

    popd
}

prepare_runc() {
    echo "[+] prepare runc"
    github_name "runc-${RUNC_VERSION}"
}

compile_runc() {
    echo "[+] compiling runc"
    make BUILDTAGS='seccomp'
}

install_runc() {
    echo "[+] install runc"
    mkdir -p "${ROOTDIR}/usr/bin"
    cp -av runc "${ROOTDIR}/usr/bin/"
}

build_runc() {
    pushd "${DISTDIR}"

    dependencies_runc
    download_runc
    extract_runc

    popd
    pushd ${RUNC_HOME}/runc

    prepare_runc
    compile_runc
    install_runc

    popd
}

