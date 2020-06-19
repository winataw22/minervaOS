LIBCAP_MUSL_PKGNAME="libcap"
LIBCAP_MUSL_VERSION="2.33"
LIBCAP_MUSL_CHECKSUM="c23bbc02b13d10c3889ef2b1bed34071"
LIBCAP_MUSL_LINK="https://git.kernel.org/pub/scm/linux/kernel/git/morgan/libcap.git/snapshot/libcap-${LIBCAP_MUSL_VERSION}.tar.gz"

download_libcap-musl() {
    download_file $LIBCAP_MUSL_LINK $LIBCAP_MUSL_CHECKSUM
}

extract_libcap-musl() {
    tar -xf ${DISTDIR}/${LIBCAP_MUSL_PKGNAME}-${LIBCAP_MUSL_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_libcap-musl() {
    echo "[+] configuring: ${LIBCAP_MUSL_PKGNAME}"

    # disable shared lib
    sed -i 's/all: $(MINLIBNAME)/all:/' libcap/Makefile
    sed -i '/0644 $(MINLIBNAME)/d' libcap/Makefile

    # disable tests
    sed -i '/$(MAKE) -C tests/d' Makefile
}

compile_libcap-musl() {
    echo "[+] compiling: ${LIBCAP_MUSL_PKGNAME}"

    make ${MAKEOPTS} CC=musl-gcc LD=musl-gcc GOLANG=no prefix=/
}

install_libcap-musl() {
    echo "[+] installing: ${LIBCAP_MUSL_PKGNAME}"

    make DESTDIR="${ROOTDIR}" GOLANG=no prefix=/ install
}

build_libcap-musl() {
    pushd ${DISTDIR}

    download_libcap-musl
    extract_libcap-musl

    popd
    pushd "${WORKDIR}/${LIBCAP_MUSL_PKGNAME}-${LIBCAP_MUSL_VERSION}"

    prepare_libcap-musl
    compile_libcap-musl
    install_libcap-musl

    popd
}

