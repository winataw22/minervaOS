LINUX_KERNEL_VERSION="4.19.86"
LINUX_KERNEL_CHECKSUM="14dbbe3ae9ae1b470aac1e1408d4ea21"
LINUX_KERNEL_LINK="https://cdn.kernel.org/pub/linux/kernel/v4.x/linux-${LINUX_KERNEL_VERSION}.tar.xz"
K3OS_REPOSITORY="https://github.com/threefoldtech/k3os"
K3OS_BRANCH="zos-patch"

dependencies_k3os() {
    apt-get install -y git build-essential bc bison flex
}

download_k3os() {
    download_file ${LINUX_KERNEL_LINK} ${LINUX_KERNEL_CHECKSUM}
    download_git ${K3OS_REPOSITORY} ${K3OS_BRANCH}
}

extract_k3os() {
    echo "[+] extracting linux-kernel"

    rm -rf ${WORKDIR}/*
    tar -xpf ${DISTDIR}/linux-${LINUX_KERNEL_VERSION}.tar.xz -C ${WORKDIR}

    echo "[+] extracting k3os code"
    cp -a k3os ${WORKDIR}/
}

prepare_k3os() {
    echo "[+] prepare k3os tools"
    github_name "k3os"
}

compile_k3os() {
    echo "[+] compile k3os linux-kernel"
    cp -v ${FILESDIR}/kernel-config linux-${LINUX_KERNEL_VERSION}/.config

    pushd linux-${LINUX_KERNEL_VERSION}
    make ${MAKEOPTS} vmlinux
    popd

    echo "[+] building k3os images"
}

install_k3os() {
    echo "[+] install k3os linux-kernel"
    cp linux-${LINUX_KERNEL_VERSION}/vmlinux ${ROOTDIR}/k3os-vmlinux

    echo "[+] install k3os images"

}

build_k3os() {
    pushd "${DISTDIR}"

    dependencies_k3os
    download_k3os
    extract_k3os

    popd
    pushd ${WORKDIR}

    prepare_k3os
    compile_k3os
    install_k3os

    popd
}

