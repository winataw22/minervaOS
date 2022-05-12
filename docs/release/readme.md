# Releases of Zero-OS

We use a simple pipeline release workflow. Building and file distribution are made using GitHub Actions.
Usable files are available on the [Zero-OS Hub](https://hub.grid.tf/tf-zos).

This pipeline is made to match the 3 different type of running mode of 0-OS. For more information head to the [upgrade documentation](../identity/upgrade.md).

## Development build

On a push to main branch on the zos repository, a new development build is triggered. If the build succeed,
binaries are packed into an flist and uploaded to the [tf-autobuilder](https://hub.grid.tf/tf-autobuilder) repository of the hub.

This flist is then promoted into the [tf-zos](https://hub.grid.tf/tf-zos) repository of the hub and a symlink to this latest build is made (`tf-autobuilder/zos:development-3:latest.flist`)

## Testing build

As soon as a version seems good to be tested using our testnet, we will produce a release within GitHub.
This release will be tagged `vX.Y.Z-something` (eg: `v3.0.4-rc3`). Like development builds, everything is compiled
and uploaded to `tf-autobuilder`. However the symlink is not created automatically to `tf-zos` repo. The linking has to be triggered manually to make sure an update of testnet is intended and won't happen by accident. The execution of the `Deploy` workflow github action has to be done with the right version (like `v3.0.4-rc3`)

## Production build

As soon as a build is bullet-proof tested and working fine, a new release will be made within GitHub and this
release will be tagged `vX.Y.Z`. This is the final versioning form.

Like testnet, manual step has to be done to create the link under `tf-zos` to be able to have full control of when to update mainnet

# Always Up-to-date

If you want to always uses the latest up-to-date build of our releases, you should uses theses files:

- Development: https://playground.hub.grid.tf/tf-autobuilder/zos:development-3:latest.flist
- Testing: https://playground.hub.grid.tf/tf-zos/zos:testing-3:latest.flist
- Production: https://playground.hub.grid.tf/tf-zos/zos:production-3:latest.flist
