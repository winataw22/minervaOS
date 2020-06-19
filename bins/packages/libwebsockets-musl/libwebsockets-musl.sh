LIBWEBSOCKETS_MUSL_PKGNAME="libwebsockets"
LIBWEBSOCKETS_MUSL_VERSION="3.2.0"
LIBWEBSOCKETS_MUSL_CHECKSUM="1d06f5602604e67e6f50cef9857c6b0c"
LIBWEBSOCKETS_MUSL_LINK="https://github.com/warmcat/libwebsockets/archive/v${LIBWEBSOCKETS_MUSL_VERSION}.tar.gz"

dependencies_libwebsockets-musl() {
    apt-get install -y cmake
}

download_libwebsockets-musl() {
    download_file $LIBWEBSOCKETS_MUSL_LINK $LIBWEBSOCKETS_MUSL_CHECKSUM ${LIBWEBSOCKETS_MUSL_PKGNAME}-${LIBWEBSOCKETS_MUSL_VERSION}.tar.gz
}

extract_libwebsockets-musl() {
    tar -xf ${DISTDIR}/${LIBWEBSOCKETS_MUSL_PKGNAME}-${LIBWEBSOCKETS_MUSL_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_libwebsockets-musl() {
    echo "[+] configuring: ${LIBWEBSOCKETS_MUSL_PKGNAME}"

    echo $ROOTDIR
    mkdir -p build && cd build
    CC="musl-gcc" cmake -DLWS_IPV6=ON \
        -DCMAKE_INSTALL_PREFIX=/ \
        -DLWS_UNIX_SOCK=ON \
        -DLWS_WITHOUT_TESTAPPS=ON \
        -DLWS_WITH_SHARED=OFF \
        -DOPENSSL_ROOT_DIR=${ROOTDIR}/../openssl-musl \
        ..
}

compile_libwebsockets-musl() {
    echo "[+] compiling: ${LIBWEBSOCKETS_MUSL_PKGNAME}"

    make ${MAKEOPTS}
}

install_libwebsockets-musl() {
    echo "[+] installing: ${LIBWEBSOCKETS_MUSL_PKGNAME}"

    make DESTDIR="${ROOTDIR}" install
}

build_libwebsockets-musl() {
    pushd "${DISTDIR}"

    dependencies_libwebsockets-musl
    download_libwebsockets-musl
    extract_libwebsockets-musl

    popd
    pushd "${WORKDIR}/${LIBWEBSOCKETS_MUSL_PKGNAME}-${LIBWEBSOCKETS_MUSL_VERSION}"

    prepare_libwebsockets-musl
    compile_libwebsockets-musl
    install_libwebsockets-musl

    popd
}
