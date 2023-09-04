# Adding a new package

Binary packages are added via providing [a build script](../../bins/), then an automated workflow will build/publish an flist with this binary.

For example, to add `rmb` binary, we need to provide a bash script with a `build_rmb` function:


```bash
RMB_VERSION="0.1.2"
RMB_CHECKSUM="4fefd664f261523b348fc48e9f1c980b"
RMB_LINK="https://github.com/threefoldtech/rmb-rs/releases/download/v${RMB_VERSION}/rmb"

download_rmb() {
    echo "download rmb"
    download_file ${RMB_LINK} ${RMB_CHECKSUM} rmb
}

prepare_rmb() {
    echo "[+] prepare rmb"
    github_name "rmb-${RMB_VERSION}"
}

install_rmb() {
    echo "[+] install rmb"

    mkdir -p "${ROOTDIR}/bin"

    cp ${DISTDIR}/rmb ${ROOTDIR}/bin/
    chmod +x ${ROOTDIR}/bin/*
}

build_rmb() {
    pushd "${DISTDIR}"

    download_rmb
    popd

    prepare_rmb
    install_rmb
}
```

Note that, you can just download a statically build binary instead of building it.

The other step is to add it to workflow to be built automatically, in [bins workflow](../../.github/workflows/bins.yaml), add your binary's job:

```yaml
jobs:
  containerd:
    ...
    ...
  rmb:
    uses: ./.github/workflows/bin-package.yaml
    with:
      package: rmb
    secrets:
      token: ${{ secrets.HUB_JWT }}
```

Once e.g. a `devnet` release is published, your package will be built then pushed to an flist repository. After that, you can start your local zos node, wait for it to finish downloading, then you should find your binary available.
