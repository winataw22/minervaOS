FIRECRACKER_BRANCH="master"
FIRECRACKER_VERSION="1eb80f8d1bbe9287cafdedebaee3fb604095fac6"
FIRECRACKER_REPOSITORY="https://github.com/firecracker-microvm/firecracker"
FIRECRACKER_LIBC="musl"
FIRECRACKER_RUST_TOOLCHAIN="1.39.0"

dependencies_firecracker() {
    # based on:
    # https://github.com/firecracker-microvm/firecracker/blob/master/tools/devctr/Dockerfile.x86_64

    apt-get -y install --no-install-recommends \
        binutils-dev cmake curl file g++ gcc gcc-aarch64-linux-gnu git \
        iperf3 iproute2 jq libcurl4-openssl-dev libdw-dev libiberty-dev \
        libssl-dev lsof make musl-tools net-tools openssh-client screen \
        pkgconf python python3 python3-dev python3-pip python3-venv zlib1g-dev

    python3 -m pip install setuptools wheel

    python3 -m pip install boto3 nsenter pycodestyle pydocstyle retry \
        pylint pytest pytest-timeout pyyaml requests requests-unixsocket

    curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain "$FIRECRACKER_RUST_TOOLCHAIN"
    . $HOME/.cargo/env

    rustup target add x86_64-unknown-linux-musl
    rustup component add rustfmt
    rustup component add clippy-preview

    cargo install cargo-kcov
    cargo install cargo-audit
    cargo kcov --print-install-kcov-sh | sh
}

download_firecracker() {
    download_git ${FIRECRACKER_REPOSITORY} ${FIRECRACKER_BRANCH}

    pushd firecracker
    echo "[+] checking out revision: ${FIRECRACKER_VERSION}"
    git checkout ${FIRECRACKER_VERSION}
    popd
}

extract_firecracker() {
    echo "[+] extracting firecracker"
    rm -rf ${WORKDIR}/*
    cp -a firecracker/* ${WORKDIR}/
}

prepare_firecracker() {
    echo "[+] prepare firecracker"
    github_name "firecracker-${FIRECRACKER_VERSION:0:8}"
}

compile_firecracker() {
    echo "[+] compile firecracker"
    target="$(uname -m)-unknown-linux-${FIRECRACKER_LIBC}"
    TARGET_CC=musl-gcc cargo build --release --target ${target}
}

install_firecracker() {
    echo "[+] install firecracker"

    mkdir -p "${ROOTDIR}/usr/bin"

    cp ${WORKDIR}/target/x86_64-unknown-linux-musl/release/firecracker ${ROOTDIR}/usr/bin/
    cp ${WORKDIR}/target/x86_64-unknown-linux-musl/release/jailer ${ROOTDIR}/usr/bin/

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_firecracker() {
    pushd "${DISTDIR}"

    dependencies_firecracker
    download_firecracker
    extract_firecracker

    popd
    pushd ${WORKDIR}

    prepare_firecracker
    compile_firecracker
    install_firecracker

    popd
}

