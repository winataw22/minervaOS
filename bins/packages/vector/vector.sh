VECTOR_VERSION="0.25.1"
VECTOR_CHECKSUM="07bcae774d8f6dc5f34a5f4f7bafd313"
VECTOR_LINK="https://github.com/vectordotdev/vector/releases/download/v${VECTOR_VERSION}/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz"
VECTOR_PACKAGE="vector.tar.gz"

download_vector() {
    download_file ${VECTOR_LINK} ${VECTOR_CHECKSUM} ${VECTOR_PACKAGE}
}

extract_vector() {
    tar -xf  ${DISTDIR}/${VECTOR_PACKAGE} -C ${WORKDIR}
}

prepare_vector() {
    echo "[+] prepare vector"
    github_name "vector-${VECTOR_VERSION}"
}

compile_vector() {
    echo "[+] compile vector"
}

install_vector() {
    echo "[+] install vector"

    mkdir -p "${ROOTDIR}/usr/bin"
    mkdir -p "${ROOTDIR}/etc/zinit"
    mkdir -p "${ROOTDIR}/etc/vector"

    cp ${WORKDIR}/vector-x86_64-unknown-linux-musl/bin/vector ${ROOTDIR}/usr/bin/vector

    cp ${FILESDIR}/zinit-vector.yaml ${ROOTDIR}/etc/zinit/vector.yaml
    cp ${FILESDIR}/vector.yaml ${ROOTDIR}/etc/vector/

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_vector() {
    pushd "${DISTDIR}"

    download_vector
    extract_vector

    popd
    pushd ${WORKDIR}

    prepare_vector
    compile_vector
    install_vector

    popd
}
