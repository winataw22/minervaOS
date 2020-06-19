OPENSSL_MUSL_PKGNAME="openssl"
OPENSSL_MUSL_VERSION="1.1.1d"
OPENSSL_MUSL_CHECKSUM="3be209000dbc7e1b95bcdf47980a3baa"
OPENSSL_MUSL_LINK="https://www.openssl.org/source/openssl-${OPENSSL_MUSL_VERSION}.tar.gz"

download_openssl-musl() {
    download_file $OPENSSL_MUSL_LINK $OPENSSL_MUSL_CHECKSUM
}

extract_openssl-musl() {
    tar -xf ${DISTDIR}/${OPENSSL_MUSL_PKGNAME}-${OPENSSL_MUSL_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_openssl-musl() {
    echo "[+] preparing: ${OPENSSL_MUSL_PKGNAME}"
    CC="musl-gcc" ./Configure --prefix=/ linux-x86_64 no-shared
}

compile_openssl-musl() {
    echo "[+] compiling: ${OPENSSL_MUSL_PKGNAME}"
    make ${MAKEOPTS}
}

install_openssl-musl() {
    echo "[+] installing: ${OPENSSL_MUSL_PKGNAME}"
    make DESTDIR="${ROOTDIR}" install_sw
}

build_openssl-musl() {
    pushd "${DISTDIR}"

    download_openssl-musl
    extract_openssl-musl

    popd
    pushd "${WORKDIR}/${OPENSSL_MUSL_PKGNAME}-${OPENSSL_MUSL_VERSION}"

    prepare_openssl-musl
    compile_openssl-musl
    install_openssl-musl

    popd
}
