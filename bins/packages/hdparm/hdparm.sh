HDPARM_VERSION="9.58"
HDPARM_CHECKSUM="4652c49cf096a64683c05f54b4fa4679"
HDPARM_LINK="https://sourceforge.net/projects/hdparm/files/hdparm/hdparm-9.58.tar.gz/download"

download_hdparm() {
    download_file ${HDPARM_LINK} ${HDPARM_CHECKSUM} hdparm-${HDPARM_VERSION}.tar.gz
}

extract_hdparm() {
    echo "[+] extracting hdparm"

    rm -rf ${WORKDIR}/*
    tar -xf ${DISTDIR}/hdparm-${HDPARM_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_hdparm() {
    echo "[+] prepare hdparm tools"
    github_name "hdparm-${HDPARM_VERSION}"
}

compile_hdparm() {
    echo "[+] compile hdparm"
    pwd
    make ${MAKEOPTS} install

    echo "[+] building hdparm images"
}

install_hdparm() {
    echo "[+] install hdparm"

    mkdir -p "${ROOTDIR}/sbin/"
    cp hdparm ${ROOTDIR}/sbin/

    echo "[+] install hdparm images"

}

build_hdparm() {
    pushd "${DISTDIR}"

    download_hdparm
    extract_hdparm

    popd
    pushd ${WORKDIR}/hdparm-${HDPARM_VERSION}

    prepare_hdparm
    compile_hdparm
    install_hdparm

    popd
}

