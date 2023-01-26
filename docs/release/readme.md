# Releases of Zero-OS

We use a simple pipeline release workflow. Building and file distribution are made using GitHub Actions.
Usable files are available on the [Zero-OS Hub](https://hub.grid.tf/tf-zos).

This pipeline is made to match the 3 different type of running mode of 0-OS. For more information head to the [upgrade documentation](../identity/upgrade.md).

## Development build

On a push to main branch on the zos repository, a new development build is triggered. If the build succeed,
binaries are packed into an flist and uploaded to the [tf-autobuilder](https://hub.grid.tf/tf-autobuilder) repository of the hub.

This flist is then promoted into the [tf-zos](https://hub.grid.tf/tf-zos) repository of the hub and a symlink to this latest build is made (`tf-autobuilder/zos:development-3:latest.flist`)

## Releases
We create 3 types of releases:
- QA release, in this release the version is suffixed by `qa<number>` for example `v3.5.0-qa1`.
- RC release, in this release the version is suffixed by `rc<number>` for example `v3.5.0-rc2`.
- Main release, is this release the version has no suffix, for example `v3.5.0`

The release cycle goes like this:
- As mentioned before devnet is updated the moment new code is available on `main` branch. Since the `dev` release is auto linked to the latest `flist` on the hub. Nodes on devnet will auto update to the latest available build.
- Creating a `qa` release, will not not trigger the same behavior on `qa` net, same for both testnet and mainnet. Instead a workflow must be triggered, this is only to make sure 100% that an update is needed.
- Once the build of the release is available, a [deploy](../../.github/workflows/grid-deploy.yaml) workflow needed to be triggered with the right version to deploy on the proper network.
  - The work flow all what it does is linking the right version under the hub [tf-zos](https://hub.grid.tf/tf-zos) repo

> The `deploy` flow is rarely used, the on chain update is also available. By setting the right version on tfchain, the link on the hub is auto-updated and hence the deploy workflow won't be needed to be triggered. Although we have it now as a safety net in case something goes wrong (chain is broken) and we need to force a specific version on ZOS.

- Development: https://playground.hub.grid.tf/tf-autobuilder/zos:development-3:latest.flist
- Testing: https://playground.hub.grid.tf/tf-zos/zos:testing-3:latest.flist
- Production: https://playground.hub.grid.tf/tf-zos/zos:production-3:latest.flist
