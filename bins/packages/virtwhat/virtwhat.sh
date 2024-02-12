# VIRTWHAT_Git
VIRTWHAT_VERSION="d163be0"
VIRTWHAT_CHECKSUM="d84c9a51e2869fbe949a83adf7f80b56"
VIRTWHAT_LINK="http://git.annexia.org/?p=virt-what.git;a=snapshot;h=${VIRTWHAT_VERSION};sf=tgz"

download_virtwhat() {
    download_file ${VIRTWHAT_LINK} ${VIRTWHAT_CHECKSUM} "virt-what-${VIRTWHAT_VERSION}.tar.gz"
}

extract_virtwhat() {
    tar -xf ${DISTDIR}/virt-what-${VIRTWHAT_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_virtwhat() {
    echo "[+] prepare virt-what"
    github_name "virtwhat-${VIRTWHAT_VERSION}"
}

compile_virtwhat() {
    echo "[+] compile virt-what"
    pushd ${WORKDIR}/virt-what-${VIRTWHAT_VERSION}
    autoreconf -i
    autoconf
    ./configure
    make
    popd
}

install_virtwhat() {
    echo "[+] install virt-what"

    mkdir -p "${ROOTDIR}/usr/bin"


    cp ${WORKDIR}/virt-what-${VIRTWHAT_VERSION}/virt-what ${ROOTDIR}/usr/bin/virt-what
    cp ${WORKDIR}/virt-what-${VIRTWHAT_VERSION}/virt-what-cpuid-helper ${ROOTDIR}/usr/bin/virt-what-cpuid-helper

    chmod +x ${ROOTDIR}/usr/bin/virt-what
    chmod +x ${ROOTDIR}/usr/bin/virt-what-cpuid-helper
}

build_virtwhat() {
    pushd "${DISTDIR}"

    download_virtwhat
    extract_virtwhat

    popd
    pushd ${WORKDIR}

    prepare_virtwhat
    compile_virtwhat
    install_virtwhat

    popd
}

