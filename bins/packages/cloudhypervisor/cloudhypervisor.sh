CLOUDHYPERVISOR_VERSION="0.13.0"
CLOUDHYPERVISOR_CHECKSUM="44bd96c255b8d573a83eef2364d934ae"
CLOUDHYPERVISOR_LINK="https://github.com/cloud-hypervisor/cloud-hypervisor/archive/v${CLOUDHYPERVISOR_VERSION}.tar.gz"
CLOUDHYPERVISOR_RUST_TOOLCHAIN="1.50.0"

dependencies_cloudhypervisor() {
    apt-get install -y curl build-essential

    curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain "${CLOUDHYPERVISOR_RUST_TOOLCHAIN}"
    . $HOME/.cargo/env

    rustup target add x86_64-unknown-linux-musl
}

download_cloudhypervisor() {
    echo "down"
    download_file ${CLOUDHYPERVISOR_LINK} ${CLOUDHYPERVISOR_CHECKSUM} cloud-hypervisor-${CLOUDHYPERVISOR_VERSION}.tar.gz
}

extract_cloudhypervisor() {
    echo "[+] extracting cloud-hypervisor"
    tar -xf ${DISTDIR}/cloud-hypervisor-${CLOUDHYPERVISOR_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_cloudhypervisor() {
    echo "[+] prepare cloud-hypervisor"
    github_name "cloud-hypervisor-${CLOUDHYPERVISOR_VERSION}"
}

compile_cloudhypervisor() {
    echo "[+] compile cloud-hypervisor"
    cargo build --release --target=x86_64-unknown-linux-musl --all
}

install_cloudhypervisor() {
    echo "[+] install cloud-hypervisor"

    mkdir -p "${ROOTDIR}/usr/bin"


    RELEASEDIR="target/x86_64-unknown-linux-musl/release"
    cp "${RELEASEDIR}/cloud-hypervisor" ${ROOTDIR}/usr/bin/

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_cloudhypervisor() {
    pushd "${DISTDIR}"

    dependencies_cloudhypervisor
    download_cloudhypervisor
    extract_cloudhypervisor

    popd
    pushd ${WORKDIR}/cloud-hypervisor-${CLOUDHYPERVISOR_VERSION}

    prepare_cloudhypervisor
    compile_cloudhypervisor
    install_cloudhypervisor

    popd
}
