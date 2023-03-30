# Introduction
This document explains in a nutshell the internals of ZOS. This includes the boot process, architecture, the internal modules (and their responsibilities), and the inter-process communication.

## Booting
ZOS is a linux based operating system in the sense that we use the main-stream linux kernel with no modifications (but heavily customized). The base image of ZOS includes linux, busybox, [zinit](https://github.com/threefoldtech/zinit) and other required tools that are needed during the boot process. The base image is also shipped with a bootstrap utility that is self-updating on boot which kick starts everything.

For more details about the ZOS base image please check [0-initramfs](https://github.com/threefoldtech/0-initramfs).

`ZOS` uses zinit as its `init` or `PID 1` process. `zinit` acts as a process manager and it takes care of starting all required services in the right order. Using simple configuration that is available under `/etc/zinit`.

The base `ZOS` image has a zinit config to start the basic services that are required for booting. These include (mainly) but are not limited to:

- internet: A very basic service that tries to connect zos to the internet as fast (and as simple) as possible (over ethernet) using dhcp. This is needed so the system can continue the boot process. Once this one succeeds, it exits and leaves node network management to the more sophisticated ZOS module `networkd` which is yet to be downloaded and started by bootstrap.
- redis: This is required by all zos modules for its IPC (inter process communication).
- bootstrap: The bootstrap process which takes care of downloading all required zos binaries and modules. This one requires the `internet` service to actually succeed.

## Bootstrap
`bootstrap` is a utility that resides on the base image. It takes care of downloading and configuring all zos main services by doing the following:
- It checks if there is a more recent version of itself available. If it exists, the process first updates itself before proceeding.
- It checks zos boot parameters (for example, which network you are booting into) as set by https://bootstrap.grid.tf/.
- Once the network is known, let's call it `${network}`. This can either be `production`, `testing`, or `development`. The proper release is downloaded as follows:
  - All flists are downloaded from one of the [hub](https://hub.grid.tf/) `tf-zos-v3-bins.dev`, `tf-zos-v3-bins.test`, or `tf-zos-v3-bins` repos. Based on the network, only one of those repos is used to download all the support tools and binaries. Those are not included in the base image because they can be updated, added, or removed.
  - The flist `https://hub.grid.tf/tf-zos/zos:${network}-3:latest.flist.md` is downloaded (note that ${network} is replaced with the actual value). This flist includes all zos services from this repository. More information about the zos modules are explained later.
  - Once all binaries are downloaded, `bootstrap` finishes by asking zinit to start monitoring the newly installed services. The bootstrap exits and will never be started again as long as zos is running.
  - If zos is restarted the entire bootstrap process happens again including downloading the binaries because ZOS is completely stateless (except for some cached runtime data that is preserved across reboots on a cache disk).

## Zinit
As mentioned earlier, `zinit` is the process manager of zos. Bootstrap makes sure it registers all zos services for zinit to monitor. This means that zinit will take care that those services are always running, and restart them if they have crashed for any reason.

## Architecture
For `ZOS` to be able to run workloads of different types it has split its functionality into smaller modules. Where each module is responsible for providing a single functionality. For example `storaged` which manages machine storages, hence it can provide low level storage capacity to other services that need it.

As an example, imagine that you want to start a `virtual machine`. For a `virtual machine` to be able to run it will require a `rootfs` image or the image of the VM itself this is normally provided via an `flist` (managed by `flistd`), then you would need an actual persistent storage (managed by `storaged`), a virtual nic (managed by `networkd`), another service that can put everything together in a form of a VM (`vmd`). Then finally a service that orchestrates all of this and translates the user request to an actual workload `provisiond`, you get the picture.

### IPC
All modules running in zos needs to be able to interact with each other. As it shows from the previous example. For example, `provision` daemon need to be able to ask `storage` daemon to prepare a virtual disk. A new `inter-process communication` protocol and library was developed to enable this with those extra features:

- Modules do not need to know where other modules live, there are no ports, and/or urls that have to be known by all services.
- A single module can run multiple versions of an API.
- Ease of development.
- Auto generated clients.

For more details about the message bus please check [zbus](https://github.com/threefoldtech/zbus)

`zbus` uses redis as a message bus, hence redis is started in the early stages of zos booting.

`zbus` allows auto generation of `stubs` which are generated clients against a certain module interface. Hence a module X can interact with a module Y by importing the generated clients and then start making function calls.

## ZOS Processes (modules)
Modules of zos are completely internal. There is no way for an external user to talk to them directly. The idea is the node exposes a public API over rmb, while internally this API can talk to internal modules over `zbus`.

Here is a list of the major ZOS modules.

- [Identity](identity/readme.md)
- [Node](node/readme.md)
- [Storage](storage/readme.md)
- [Network](network/readme.md)
- [Flist](flist/readme.md)
- [Container](container/readme.md)
- [VM](vmd/readme.md)
- [Provision](provision/readme.md)

