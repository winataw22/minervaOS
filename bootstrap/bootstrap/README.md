# Bootstrap

Bootstrap is a multi stage strategy to bring the node to a final working
state with latest released version of everything!

## Stages

`bootstrap` to make sure it runs everything with the correct version it
will do a multiple stage bootstrap. Currently this is only two stages:

- Update self (bootstrap binary itself)
- Update software
  - Core utils and daemons
  - ZOS daemons
  - Start daemons

## How to works

- Bootstrap is used by [0-initramfs](https://github.com/threefoldtech/0-initramfs/blob/development-zos-v3/packages/modules.sh) to basically add `internet` and `bootstrap` services to the base image
- After internet service is fully started, bootstrap will start to download flists needed for zos node to work properly
- As described above bootstrap run in two stages:
  - The first stage is used to update bootstrap itself, and it is done like that to avoid re-building the image if we only changed the bootstrap code. this update is basically done from `tf-autobuilder` repo in the [hub/tf-autobuilder](https://hub.grid.tf/tf-autobuilder) and download the latest bootstrap flist
  - For the second stage bootstrap will download the flists for that env. bootstrap cares about `runmode` argument that we pass during the start of the node. for example if we passed `runmode=dev` it will get the the tag `development` under [hub/tf-zos](https://hub.grid.tf/tf-zos) each tag is linked to a sub-directory where all flists for this env exists to be downloaded and installed on the node

## Testing in Developer setup

To test bootstrap changes on a local dev-setup you need to do the following

- under zos/qemu `cp -r overlay.normal overlay.custom`
- build `bootstrap` bin
- copy the `bootstrap` bin to overlay.custom/sbin/
- remove dir `overlay.custom/bin`
- remove all files under `overlay.custom/etc/zinit/`
- add the file overlay.custom/etc/zinit/bootstrap.yaml with the following content

```
exec: bootstrap -d
oneshot: true
after:
  - internet
```

- remove overlay link under `qemu/overlay `
- create a new link pointing to overlay.custom under zos/qemu `ln -s overlay.custom overlay`
- boot your vm as normal
