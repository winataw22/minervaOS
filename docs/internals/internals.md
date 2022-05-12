# Introduction
This document explains in a nutshell the internals of ZOS. This include the boot process, architecture, the internal modules (and their responsibilities), and the inter-process communication.

## Booting
ZOS is a linux based operating system in the sense that we use the main-stream linux kernel with no modifications (but heavily customized). The base image of ZOS include linux, busybox, [zinit](https://github.com/threefoldtech/zinit) and other required tools that are needed during the boot process. The base image also is shipped with a bootstrap utility that is self-updating on boot which kick starts everything.

For more details about ZOS base image please check [0-initramfs](https://github.com/threefoldtech/0-initramfs).

`ZOS` uses zinit as it's `init` or `PID 1` process. `zinit` acts as a process manager and it takes care of starting all required services in the right order. using simple configuration that is available under `/etc/zinit`

The base `ZOS` image has few zinit config to start the basic service that are required for booting. This include (mainly) but not limited to

- internet: A very basic service that tries to connect zos to the internet as fast (and as simple) as possible (over ethernet) using dhcp. This is needed so the system can continue the boot process. Once this one succeeds, it exits and leaves node network management to the more sophisticated ZOS module `networkd` which is yet to be downloaded and started by bootstrap.
- redis: This is required by all zos modules for it's IPC (inter process communication)
- bootstrap: The bootstrap process which takes care of downloading all required zos binaries and modules. This one required `internet` service to actually succeed

## Bootstrap
`bootstrap` is utility that resides on the base image. It takes care of downloading and configuring all zos main services by doing the following:
- It checks if there a more recent version of itself available. If exists, the process first update itself before proceeding
- It checks zos boot parameters (for example, which network you are booting into) as set by https://bootstrap.grid.tf/.
- Once network is know let's call it `${network}`. This can either be `production`, `testing`, or `development`. The proper release is downloaded as follows:
  - All flists are downloaded from one of the [hub](https://hub.grid.tf/) `tf-zos-v3-bins.dev`, `tf-zos-v3-bins.test`, or `tf-zos-v3-bins` repos. Based on the network, only one of those repos is used to download all the support tools and binaries. Those are not included in the base image because they can be updated, added, or removed.
  - The flist `https://hub.grid.tf/tf-zos/zos:${network}-3:latest.flist.md` is downloaded (note that ${network} is replaced with actual value). This flist include all zos services from this repository. More information about the zos modules are explained later.
  - Once all binaries are downloaded, `bootstrap` finishes by asking zinit to start monitoring the newly installed services. The bootstrap exits and never to be started again as long as zos is running.
  - If zos is restarted the entire bootstrap process happens again including the binaries download because ZOS is completely stateless (except for some cached runtime data that is preserved across reboots on a cache disk)

## Zinit
As mentioned earlier, `zinit` is the process manager of zos. Bootstrap make sure it registers all zos services for zinit to monitor. This means that zinit will take care that those services are always running even, and restarted if they crashed for any reason.

## Architecture
for `ZOS` to be able to run workloads of different types it has split it's functionality into smaller modules. Where each module is responsible of providing a single functionality. For example `storaged` which manages machine storages, hence it can provide low level storage capacity to other services that it needs it.

As an example, imaging you want to start a `virtual machine`. For a `virtual machine` to be able to run it will require a `rootfs` image or the image of the VM itself this is normally provided via an `flist` (managed by `flistd`), then you would need an actual persisted storage (managed by `storaged`), a virtual nic (managed by `networkd`), another service that can put everything together in a form of a VM (`vmd`). Then finally a service that orchestrate all of this and translate the user request to an actual workload `provisiond`, you get the picture.

### IPC
All modules running in zos needs to be able to interact with each other. As it shows from the previous example. For example, `provision` daemon need to be able to ask `storage` daemon to prepare a virtual disk. A new `inter-process communication` protocol and library was developed to enable this with those extra features:

- Modules do not need to know where other modules live, there is no ports, and/or urls that has to be know by all services.
- A single module can run multiple versions of an API
- Ease of development
- Auto generated clients.

For more details over the message bus please check [zbus](https://github.com/threefoldtech/zbus)

`zbus` uses redis as a message bus, hence redis is started in the early stages of zos booting.

`zbus` allows auto generation of `stubs` which are generated clients against a certain module interface. Hence a module X can interact with module Y by importing the generated clients and then start making function calls.

## ZOS Processes (modules)
Here you are a list of the major ZOS modules.

- [Identity](identity/readme.md)
- [Node](node/readme.md)
- [Storage](storage/readme.md)
- [Network](network/readme.md)
- [Flist](flist/readme.md)
- [Container](container/readme.md)
- [VM](vmd/readme.md)
- [Provision](provision/readme.md)
