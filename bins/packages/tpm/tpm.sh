TPM_VERSION="5.2"
TPM_CHECKSUM="0057615ef43b9322d4577fc3bde0e8d6"
TPM_LINK="https://github.com/tpm2-software/tpm2-tools/releases/download/${TPM_VERSION}/tpm2-tools-${TPM_VERSION}.tar.gz"

TSS_VERSION="3.2.0"
TSS_CHECKSUM="0d60d0df3fd0daae66881a3022281323"
TSS_LINK="https://github.com/tpm2-software/tpm2-tss/releases/download/${TSS_VERSION}/tpm2-tss-${TSS_VERSION}.tar.gz"

download_tss() {
    download_file $TSS_LINK $TSS_CHECKSUM
}

extract_tss() {
    if [ ! -d "tpm2-tss-${TSS_VERSION}" ]; then
        echo "[+] extracting: tpm2-tss-${TSS_VERSION}"
        tar -xf ${DISTDIR}/tpm2-tss-${TSS_VERSION}.tar.gz -C ${WORKDIR}
    fi
}

prepare_tss() {
    echo "[+] preparing tpm-tss"
    apt -y update
    apt -y install \
    autoconf-archive \
    libcmocka0 \
    libcmocka-dev \
    procps \
    iproute2 \
    build-essential \
    git \
    pkg-config \
    gcc \
    libtool \
    automake \
    libssl-dev \
    uthash-dev \
    autoconf \
    doxygen \
    libjson-c-dev \
    libini-config-dev \
    libcurl4-openssl-dev \
    libltdl-dev
    
    ./configure --prefix=/usr 
}

compile_tss() {
    echo "[+] compiling tpm-tss"
    make ${MAKEOPTS}
}

install_tss() {
    echo "[+] installing tpm-tss"
    make DESTDIR="${ROOTDIR}" install
}

download_tpm() {
    download_file $TPM_LINK $TPM_CHECKSUM
}

extract_tpm() {
    if [ ! -d "tpm2-tools-${TPM_VERSION}" ]; then
        echo "[+] extracting: tpm2-tools-${TPM_VERSION}"
        tar -xf ${DISTDIR}/tpm2-tools-${TPM_VERSION}.tar.gz -C ${WORKDIR}
    fi
}

prepare_tpm() {
    echo "[+] preparing tpm"
    github_name "tpm-${TPM_VERSION}"
    ./configure --prefix=/usr 
}

compile_tpm() {
    echo "[+] compiling tpm"
    make ${MAKEOPTS}
}

install_tpm() {
    echo "[+] installing tpm"
    make DESTDIR="${ROOTDIR}" install
}

build_tpm() {
    pushd "${DISTDIR}"
    download_tss
    extract_tss
    popd
    
    pushd "${WORKDIR}/tpm2-tss-${TSS_VERSION}"

    prepare_tss
    compile_tss
    install_tss

    popd

    pushd "${DISTDIR}"
    download_tpm
    extract_tpm
    popd
    
    pushd "${WORKDIR}/tpm2-tools-${TPM_VERSION}"

    export PKG_CONFIG_PATH="${ROOTDIR}/usr/lib/pkgconfig/"
    export CFLAGS="-I${ROOTDIR}/usr/include"
    export LDFLAGS="-L${ROOTDIR}/usr/lib"

    prepare_tpm
    compile_tpm
    install_tpm

    unset PKG_CONFIG_PATH
    unset CFLAGS
    unset LDFLAGS

    popd

    clean_up
}

clean_up(){
    pwd
    pushd releases/tpm
    rm -rf lib/*.a
    rm -rf lib/*.la
    rm -rf etc/init.d
    rm -rf usr/lib/*.a
    rm -rf usr/lib/*.la
    rm -rf usr/share/doc
    rm -rf usr/share/gtk-doc
    rm -rf usr/share/man
    rm -rf usr/share/locale
    rm -rf usr/share/info
    rm -rf usr/share/bash-completion
    rm -rf usr/lib/pkgconfig
    rm -rf usr/include
    popd
}