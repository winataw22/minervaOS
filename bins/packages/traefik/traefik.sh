TRAEFIK_VERSION="2.9.9"
TRAEFIK_CHECKSUM="83d21fc65ac5f68df985c4dca523e590"
TRAEFIK_LINK="https://github.com/traefik/traefik/releases/download/v${TRAEFIK_VERSION}/traefik_v${TRAEFIK_VERSION}_linux_amd64.tar.gz"
TRAEFIK_PACKAGE="vector.tar.gz"

download_traefik() {
    echo "download traefik"
    download_file ${TRAEFIK_LINK} ${TRAEFIK_CHECKSUM} ${TRAEFIK_PACKAGE}
}

extract_traefik() {
    tar -xf  ${DISTDIR}/${TRAEFIK_PACKAGE} -C ${WORKDIR}
}

prepare_traefik() {
    echo "[+] prepare traefik"
    github_name "traefik-${TRAEFIK_VERSION}"
}

install_traefik() {
    echo "[+] install traefik"

    mkdir -p "${ROOTDIR}"

    cp ${WORKDIR}/traefik ${ROOTDIR}/
    chmod +x ${ROOTDIR}/*
    strip ${ROOTDIR}/*
}

build_traefik() {
    pushd "${DISTDIR}"
    download_traefik
    extract_traefik
    popd

    prepare_traefik
    install_traefik
}
