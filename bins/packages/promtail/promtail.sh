PROMTAIL_VERSION="2.4.1"
PROMTAIL_CHECKSUM="6db692f6e80d2b3d41b4b16583d46dea"
PROMTAIL_LINK="https://github.com/grafana/loki/releases/download/v${PROMTAIL_VERSION}/promtail-linux-amd64.zip"

dependencies_promtail() {
    apt-get install -y unzip
}

download_promtail() {
    download_file ${PROMTAIL_LINK} ${PROMTAIL_CHECKSUM}
}

extract_promtail() {
    unzip -u ${DISTDIR}/promtail-linux-amd64.zip -d ${WORKDIR}
}

prepare_promtail() {
    echo "[+] prepare promtail"
    github_name "promtail-${PROMTAIL_VERSION}"
}

compile_promtail() {
    echo "[+] compile promtail"
}

install_promtail() {
    echo "[+] install promtail"

    mkdir -p "${ROOTDIR}/usr/bin"
    mkdir -p "${ROOTDIR}/etc/zinit"
    mkdir -p "${ROOTDIR}/etc/promtail"

    cp ${WORKDIR}/promtail-linux-amd64 ${ROOTDIR}/usr/bin/promtail

    cp ${FILESDIR}/zinit-promtail.yaml ${ROOTDIR}/etc/zinit/promtail.yaml
    cp ${FILESDIR}/promtail.yaml ${ROOTDIR}/etc/promtail/

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_promtail() {
    pushd "${DISTDIR}"

    dependencies_promtail
    download_promtail
    extract_promtail

    popd
    pushd ${WORKDIR}

    prepare_promtail
    compile_promtail
    install_promtail

    popd
}
