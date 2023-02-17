RUNC_VERSION="1.1.4"
RUNC_CHECKSUM="ae8e2ac9335b8606eeccd2e7c031350a"
RUNC_LINK="https://github.com/opencontainers/runc/archive/refs/tags/v${RUNC_VERSION}.tar.gz"

dependencies_runc() {
    apt-get install -y btrfs-progs libbtrfs-dev libseccomp-dev build-essential pkg-config

    if [ -z $GOPATH ] || [ ! -z $CI ]; then
        if command -v go > /dev/null && [ ! -z $CI ]; then
            export GOPATH=$(go env GOPATH)
        else
            curl -L https://go.dev/dl/go1.20.1.linux-amd64.tar.gz > /tmp/go1.20.1.linux-amd64.tar.gz
            tar -C /usr/local -xzf /tmp/go1.20.1.linux-amd64.tar.gz
            mkdir -p /gopath

            export PATH=/usr/local/go/bin:$PATH
        fi
    fi

    RUNC_HOME="/src/github.com/opencontainers"
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
