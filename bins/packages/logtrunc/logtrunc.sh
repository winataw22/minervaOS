LOGTRUNC_VERSION="0.1"
LOGTRUNC_CHECKSUM="0fbc114c8becf012cf9bb6c734dbc3d9"
LOGTRUNC_LINK="https://github.com/maxux/logtrunc/archive/v${LOGTRUNC_VERSION}.tar.gz"

dependencies_logtrunc() {
    apt-get install -y build-essential
}

download_logtrunc() {
    download_file ${LOGTRUNC_LINK} ${LOGTRUNC_CHECKSUM} logtrunc-${LOGTRUNC_VERSION}.tar.gz
}

extract_logtrunc() {
    tar -xf ${DISTDIR}/logtrunc-${LOGTRUNC_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_logtrunc() {
    echo "[+] prepare logtrunc"
    github_name "logtrunc-${LOGTRUNC_VERSION}"
}

compile_logtrunc() {
    echo "[+] compile promtail"
    pwd
    make release
}

install_logtrunc() {
    echo "[+] install logtrunc"

    mkdir -p "${ROOTDIR}/usr/bin"
    mkdir -p "${ROOTDIR}/etc/zinit"

    cp logtrunc ${ROOTDIR}/usr/bin/logtrunc
    cp logtrunc.yaml ${ROOTDIR}/etc/zinit/

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_logtrunc() {
    pushd "${DISTDIR}"

    dependencies_logtrunc
    download_logtrunc
    extract_logtrunc

    popd
    pushd ${WORKDIR}/logtrunc-${LOGTRUNC_VERSION}

    prepare_logtrunc
    compile_logtrunc
    install_logtrunc

    popd
}

