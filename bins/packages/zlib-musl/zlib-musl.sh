ZLIB_MUSL_PKGNAME="zlib"
ZLIB_MUSL_VERSION="1.2.12"
ZLIB_MUSL_CHECKSUM="28687d676c04e7103bb6ff2b9694c471"
ZLIB_MUSL_LINK="https://www.zlib.net/zlib-${ZLIB_MUSL_VERSION}.tar.xz"

download_zlib-musl() {
    download_file $ZLIB_MUSL_LINK $ZLIB_MUSL_CHECKSUM
}

extract_zlib-musl() {
    tar -xvf ${DISTDIR}/${ZLIB_MUSL_PKGNAME}-${ZLIB_MUSL_VERSION}.tar.xz -C ${WORKDIR}
}

prepare_zlib-musl() {
    echo "[+] configuring: ${ZLIB_MUSL_PKGNAME}"

    CC="musl-gcc" CFLAGS="-fPIC" ./configure --prefix /
}

compile_zlib-musl() {
    echo "[+] compiling: ${ZLIB_MUSL_PKGNAME}"

    make ${MAKEOPTS}
}

install_zlib-musl() {
    echo "[+] installing: ${ZLIB_MUSL_PKGNAME}"

    make DESTDIR="${ROOTDIR}" install
}

build_zlib-musl() {
    pushd "${DISTDIR}"

    download_zlib-musl
    extract_zlib-musl

    echo $WORKDIR

    popd
    pushd "${WORKDIR}/${ZLIB_MUSL_PKGNAME}-${ZLIB_MUSL_VERSION}"

    prepare_zlib-musl
    compile_zlib-musl
    install_zlib-musl

    popd
}

