FIRECRACKER_VERSION="0.22.0"
FIRECRACKER_CHECKSUM="8f1d1c70a3416db7bb8f1d05a7c6cf0d"
FIRECRACKER_LINK="https://github.com/firecracker-microvm/firecracker/archive/v${FIRECRACKER_VERSION}.tar.gz"
FIRECRACKER_LIBC="musl"
FIRECRACKER_RUST_TOOLCHAIN="1.43.1"

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

    curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain "${FIRECRACKER_RUST_TOOLCHAIN}"
    . $HOME/.cargo/env

    rustup target add x86_64-unknown-linux-musl
    rustup component add rustfmt
    rustup component add clippy-preview

    cargo install cargo-kcov
    cargo install cargo-audit
    cargo kcov --print-install-kcov-sh | sh
}

download_firecracker() {
    echo "down"
    download_file ${FIRECRACKER_LINK} ${FIRECRACKER_CHECKSUM} firecracker-${FIRECRACKER_VERSION}.tar.gz
}

extract_firecracker() {
    echo "[+] extracting firecracker"
    tar -xf ${DISTDIR}/firecracker-${FIRECRACKER_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_firecracker() {
    echo "[+] prepare firecracker"
    github_name "firecracker-${FIRECRACKER_VERSION}"
}

compile_firecracker() {
    echo "[+] compile firecracker"
    target="$(uname -m)-unknown-linux-${FIRECRACKER_LIBC}"
    TARGET_CC=musl-gcc cargo build --release --target ${target}
}

install_firecracker() {
    echo "[+] install firecracker"

    mkdir -p "${ROOTDIR}/usr/bin"


    RELEASEDIR="build/cargo_target/x86_64-unknown-linux-musl/release/"
    cp "${RELEASEDIR}/firecracker" ${ROOTDIR}/usr/bin/
    cp "${RELEASEDIR}/jailer" ${ROOTDIR}/usr/bin/

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_firecracker() {
    pushd "${DISTDIR}"

    dependencies_firecracker
    download_firecracker
    extract_firecracker

    popd
    pushd ${WORKDIR}/firecracker-${FIRECRACKER_VERSION}

    prepare_firecracker
    compile_firecracker
    install_firecracker

    popd
}

