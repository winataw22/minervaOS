GOLANG_VERSION="1.14.2"
GOLANG_LINK="https://dl.google.com/go/go${GOLANG_VERSION}.linux-amd64.tar.gz"

download_golang() {
    curl -L ${GOLANG_LINK} > go${GOLANG_VERSION}.linux-amd64.tar.gz
}

install_golang() {
    echo "[+] install golang"
    tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz
}

build_golang() {
    if [ -z $GOPATH ]; then
        if command -v go > /dev/null; then
            export GOPATH=$(go env GOPATH)
            return
        fi
    fi

    pushd "${DISTDIR}"

    download_golang
    install_golang

    mkdir -p /gopath
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=/gopath

    popd
}

