VIRTWHAT_VERSION="1.20"
VIRTWHAT_CHECKSUM="881bf446bc922eb1c702cdd2302b4459"
VIRTWHAT_LINK="https://people.redhat.com/~rjones/virt-what/files/virt-what-${VIRTWHAT_VERSION}.tar.gz"

download_virtwhat() {
    download_file ${VIRTWHAT_LINK} ${VIRTWHAT_CHECKSUM}
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

