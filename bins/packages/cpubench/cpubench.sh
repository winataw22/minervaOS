CPU_BENCHMARK_VERSION="v0.1"
CPU_BENCHMARK_CHECKSUM="25891eb15ec0b1bb8d745a8af3907895"
CPU_BENCHMARK_LINK="https://github.com/threefoldtech/cpu-benchmark-simple/releases/download/${CPU_BENCHMARK_VERSION}/grid-cpubench-simple-0.1-linux-amd64-static"

download_cpubench() {
    echo "downloading cpubench"
    download_file ${CPU_BENCHMARK_LINK} ${CPU_BENCHMARK_CHECKSUM} cpubench
}


prepare_cpubench() {
    echo "[+] prepare cpubench"
    github_name "cpubench-${CPU_BENCHMARK_VERSION}"
}

install_cpubench() {
    echo "[+] install cpubench"
    mkdir -p "${ROOTDIR}/usr/bin"

    cp ${DISTDIR}/cpubench ${ROOTDIR}/usr/bin/cpubench
    chmod +x ${ROOTDIR}/usr/bin/cpubench
}

build_cpubench() {
    pushd "${DISTDIR}"

    download_cpubench
    prepare_cpubench
    install_cpubench

    popd
}
