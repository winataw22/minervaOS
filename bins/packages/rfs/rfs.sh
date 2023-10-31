RFS_VERSION_V1="1.1.1"
RFS_VERSION_V2="2.0.3"
RFS_CHECKSUM_V1="974b8dc45ae9c1b00238a79b0f4fc9de"
RFS_CHECKSUM_V2="9257a347ace72923a84e90ceb06b4105"
RFS_LINK_V1="https://github.com/threefoldtech/rfs/releases/download/v${RFS_VERSION_V1}/rfs"
RFS_LINK_V2="https://github.com/threefoldtech/rfs/releases/download/v${RFS_VERSION_V2}/rfs"

download_rfs() {
    download_file ${RFS_LINK_V1} ${RFS_CHECKSUM_V1} rfs-${RFS_VERSION_V1}
    download_file ${RFS_LINK_V2} ${RFS_CHECKSUM_V2} rfs-${RFS_VERSION_V2}
}

prepare_rfs() {
    echo "[+] prepare rfs"
    github_name "rfs-${RFS_VERSION_V2}"
}

install_rfs() {
    echo "[+] install rfs"

    mkdir -p "${ROOTDIR}/sbin"

    cp -av rfs-${RFS_VERSION_V1} "${ROOTDIR}/sbin/g8ufs"
    chmod +x "${ROOTDIR}/sbin/g8ufs"

    cp -av rfs-${RFS_VERSION_V2} "${ROOTDIR}/sbin/rfs"
    chmod +x "${ROOTDIR}/sbin/rfs"

}

build_rfs() {
    pushd "${DISTDIR}"

    download_rfs
    prepare_rfs
    install_rfs

    popd
}
