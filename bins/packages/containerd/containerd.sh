CONTAINERD_VERSION="1.6.18"
CONTAINERD_CHECKSUM="1ac525600fe7ba6ef76cf8a833153768"
CONTAINERD_LINK="https://github.com/containerd/containerd/archive/refs/tags/v${CONTAINERD_VERSION}.tar.gz"

dependencies_containerd() {
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

    CONTAINERD_HOME="/src/github.com/containerd"
}

download_containerd() {
    download_file ${CONTAINERD_LINK} ${CONTAINERD_CHECKSUM} containerd-${CONTAINERD_VERSION}.tar.gz
}

extract_containerd() {
    mkdir -p ${CONTAINERD_HOME}
    rm -rf ${CONTAINERD_HOME}/containerd

    pushd ${CONTAINERD_HOME}

    echo "[+] extracting: containerd-${CONTAINERD_VERSION}"
    tar -xf ${DISTDIR}/containerd-${CONTAINERD_VERSION}.tar.gz -C .
    mv containerd-${CONTAINERD_VERSION} containerd

    popd
}

prepare_containerd() {
    echo "[+] prepare containerd"
    github_name "containerd-${CONTAINERD_VERSION}"
}

compile_containerd() {
    echo "[+] compiling containerd"
    make CGO_CFLAGS=-I${ROOTDIR}/usr/include
}

install_containerd() {
    echo "[+] install containerd"

    mkdir -p "${ROOTDIR}/usr/bin"
    mkdir -p "${ROOTDIR}/etc/containerd"
    mkdir -p "${ROOTDIR}/etc/zinit"

    cp -av bin/* "${ROOTDIR}/usr/bin/"
    cp -av ${FILESDIR}/config.toml "${ROOTDIR}/etc/containerd/"
    cp -av ${FILESDIR}/containerd.yaml "${ROOTDIR}/etc/zinit/"
}

build_containerd() {
    pushd "${DISTDIR}"

    dependencies_containerd
    download_containerd
    extract_containerd

    popd
    pushd ${CONTAINERD_HOME}/containerd

    prepare_containerd
    compile_containerd
    install_containerd

    popd
}
