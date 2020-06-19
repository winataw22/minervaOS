JSONC_MUSL_PKGNAME="json-c"
JSONC_MUSL_VERSION="0.13.1-20180305"
JSONC_MUSL_CHECKSUM="20dba7bf773599a0842745a2fe5b7cd3"
JSONC_MUSL_LINK="https://github.com/json-c/json-c/archive/json-c-${JSONC_MUSL_VERSION}.tar.gz"

download_libjsonc-musl() {
    download_file $JSONC_MUSL_LINK $JSONC_MUSL_CHECKSUM
}

extract_libjsonc-musl() {
    tar -xf ${DISTDIR}/${JSONC_MUSL_PKGNAME}-${JSONC_MUSL_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_libjsonc-musl() {
    echo "[+] configuring: ${JSONC_MUSL_PKGNAME}"

    CC="musl-gcc" ./configure --disable-shared --enable-static --prefix /
}

compile_libjsonc-musl() {
    echo "[+] compiling: ${JSONC_MUSL_PKGNAME}"

    make ${MAKEOPTS}
}

install_libjsonc-musl() {
    echo "[+] installing: ${JSONC_MUSL_PKGNAME}"

    make DESTDIR="${ROOTDIR}" install
}

build_libjsonc-musl() {
    pushd "${DISTDIR}"

    download_libjsonc-musl
    extract_libjsonc-musl

    popd
    pushd "${WORKDIR}/${JSONC_MUSL_PKGNAME}-${JSONC_MUSL_PKGNAME}-${JSONC_MUSL_VERSION}"

    prepare_libjsonc-musl
    compile_libjsonc-musl
    install_libjsonc-musl

    popd
}
