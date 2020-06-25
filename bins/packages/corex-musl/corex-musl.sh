COREX_MUSL_PKGNAME="corex"
COREX_MUSL_VERSION="2.1.3"
COREX_MUSL_CHECKSUM="6d94357503fe123aac8c22b86c004bca"
COREX_MUSL_LINK="https://github.com/threefoldtech/corex/archive/${COREX_MUSL_VERSION}.tar.gz"

dependencies_corex-musl() {
    apt-get install -y xxd
}

download_corex-musl() {
    download_file ${COREX_MUSL_LINK} ${COREX_MUSL_CHECKSUM} corex-${COREX_MUSL_VERSION}.tar.gz
}

extract_corex-musl() {
    tar -xf ${DISTDIR}/${COREX_MUSL_PKGNAME}-${COREX_MUSL_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_corex-musl() {
    echo "[+] preparing: corex"
    github_name "corex-${COREX_MUSL_VERSION}"
}

compile_corex-musl() {
    echo "[+] compiling: corex"
    pushd src

    PKDIR="${ROOTDIR}/.."

    ZLIB_INCLUDE="-I${PKDIR}/zlib-musl/include"
    LIBWEBSOCKETS_INCLUDE="-I${PKDIR}/libwebsockets-musl/include"
    LIBJSONC_INCLUDE="-I${PKDIR}/libjsonc-musl/include/json-c"
    OPENSSL_INCLUDE="-I${PKDIR}/openssl-musl/include"
    LIBCAP_INCLUDE="-I${PKDIR}/libcap-musl/include"

    ZLIB_LIBS="-L${PKDIR}/zlib-musl/lib"
    LIBWEBSOCKETS_LIBS="-L${PKDIR}/libwebsockets-musl/lib"
    LIBJSONC_LIBS="-L${PKDIR}/libjsonc-musl/lib"
    OPENSSL_LIBS="-L${PKDIR}/openssl-musl/lib"
    LIBCAP_LIBS="-L${PKDIR}/libcap-musl/lib"

    export CFLAGS="$ZLIB_INCLUDE $LIBWEBSOCKETS_INCLUDE $LIBJSONC_INCLUDE $OPENSSL_INCLUDE $LIBCAP_INCLUDE"
    export LDFLAGS="$ZLIB_LIBS $LIBWEBSOCKETS_LIBS $LIBJSONC_LIBS $OPENSSL_LIBS $LIBCAP_LIBS"

    CC="musl-gcc" make ${MAKEOPTS}

    unset CFLAGS
    unset LDFLAGS

    popd
}

install_corex-musl() {
    echo "[+] installing: corex"
    mkdir -p "${ROOTDIR}/usr/bin"
    cp -avL src/corex "${ROOTDIR}/usr/bin/"
}

build_corex-musl() {
    pushd "${DISTDIR}"

    dependencies_corex-musl
    download_corex-musl
    extract_corex-musl

    popd
    pushd "${WORKDIR}/${COREX_MUSL_PKGNAME}-${COREX_MUSL_VERSION}"

    prepare_corex-musl
    compile_corex-musl
    install_corex-musl

    popd
}
