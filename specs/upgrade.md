# Introduction
The purpose of this document is to discuss how zos should do the following:
- Small image size
- Booting always with the latest greatest binaries
- Live update of a running ZOS instance.
- We don't need to publish an update in different formats (image, and or flist for live update). So it doesn't matter if you starting a new machine or doing a live update.
- Seamless data migration

## Current status is
- We have a single 0-OS image that is updated manually by triggering a new build whenever we feel like it
- The live-upgrading is done using upgraded ([see doc](https://github.com/threefoldtech/zos/tree/master/docs/upgrade)) and we have a development version of the "upgrade server" at https://versions.dev.grid.tf. This server list all the available version and detail of the flist to use to apply the upgrade

Now with what we have at the moment there is already multiple issues:
1. there is no synchronization between the version publish for upgraded and the 0-OS images
2. If there are some migration steps to be done, only the live-upgrade currently support it. There is nothing in place in the 0-OS images that allow to do any migration.
3. As new version are published and if we don't creates more recent images, the time to apply all update at boot will become more and more long.
4. If we always starts from an old 0-OS image and redo the full incremental update, we will try to redo the same data migration multiple time, which will eventually create a disaster...

# Proposal
## Definition

### files manifest
This is a description of a set of files, that describes the following:
- Where the files should be placed relative to a base path (for example `rootfs` of the system)
- The hash of the files for validation of the binaries
- A manifest checksum with a digital signature which grantees the authenticity of the manifest

Currently the `flist` format implements most of those requirements. Other text based manifest files can be used instead to avoid having a dependency on 0-fs during the boot (hence requirement for a sufficient cache)

### zos image
An image is the minimal software that is required for a node to boot. This includes (but not limited to)
- linux kernel
- init (zinit)
- busybox
- various open source libraries and services that needed during the runtime of the machine

> For more information please check the `0-initramfs` repo

Currently we need to rebuild this image every time we have an update to one of our daemons, which we will try to change after implementing this document.

### bootstrap
A bootstrap program is an executable that make sure to prepare the machine for starting, this can include downloading files and writing down configuration files. The boostrap should exit once the machine is setup correctly and never run again during the life time of the machine.

## Booting new node
The following procedure is what is needed to boot a new machine to `zos:v2`

- Start the node with the current available zos image (over ipxe or usb)
- Once the kernel hands over control to `init` the init will only run the `bootstrap` program (of course also basic services)
- The bootstrap will download the latest [manifest](#files-manifest)
- The bootstrap will then download the manifest files (or if using 0-fs, just mount the flist) then make sure that the files are linked in proper locations.
- The bootstrap last job before it exits is to make sure `zinit monitor` is called on all unit files.

> Since the bootstrap will require extra storage we have 2 suggestion:
> - Either the storage modules is built-in the image, and make sure cache and other disks are mounted. (i don not like this option)
> - Create another mem disk of proper size, use it as a temporary cache for the downloads, then cleared up after the files are copied to the tmpfs of the rootfs

## Data Migration
Since some daemons need to keep some sort of state on disk. Daemons **MUST** make sure they are always compatible with the older version of their data. It's totally up to the daemon to decide how it's gonna track the version number of the data schema, but we provided a small util library to work with [versioned](../modules/versioned) data.

> We are concerned with the version of the data `schema` not the version of the data object.

## Live Migration
The proposal does not change much of what the live migration does at the moment, except it must be aware that the manifest now includes *ALL* the files, not only updated one. So we need to take care not to restart services that does not need to.

The live migration, can mount the flist with new files, copy everything in place and apply config (if needed). Daemons that requires restart can be restarted.

No data migration scripts are needed, since it's up to the daemon to take care of its own data as explained above.
