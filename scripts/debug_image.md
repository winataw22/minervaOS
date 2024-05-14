# Debug Image Script Documentation

## Purpose

The debug image script facilitates development and debugging of flists runs on ZOS. It ensures that the image is bootable with the provided configurations, making it a valid cloud image.

## Usage

Either during development, specify a directory containing the rootfs. Or to debug existing flist, pass an flist url.

```bash
# run image from a directory
./debug_image.sh --image /tmp/rootfs

# run image from a directory with a create cidata image
./debug_image.sh --image /tmp/rootfs --debug true --cidata /tmp/cloud-init.img

# run image from an flist with login cred. `foo:bar`
./debug_image.sh --image https://hub.grid.tf/tf-official-apps/discourse-v4.0.flist --init /start.sh --user foo --pass bar
```

## Image Types

- Machine: Includes the full rootfs with a kernel and initramfs image.
    the creation tutorial [here](../docs/manual/zmachine/zmachine.md)

- Container: Contains only the rootfs.
    in this case wil use kernel and initramfs from cloud container [flist](https://hub.grid.tf/tf-autobuilder/cloud-container-9dba60e.flist.md)

## Flags

- `--image`: [REQUIRED] directory or flist url
- `--d`: enables `set -x` in the bash script
- `--h`: show the help message
- `--c`: run the image in container mode, will provide kernel/initrd from cloud-container
- `--kernel`: kernel file path (compressed or uncompressed). default: `<rootfs>/boot/vmlinuz`
- `--initramfs`: Initrd image path. default: `<rootfs>/boot/initrd.img`
- `--init`: entrypoint for the machine.
- `--cidata`: optional cloud init image. will create one at `/tmp/cidata.img` if not provided
- `--user`: cloud-init username. default is user
- `--pass`: cloud-init password. default is pass
- `--name`: cloud-init machine name. default is cloud

NOTE:

- if you are passing a path to rootfs directory to `--image` flag, make sure it doesn't contain `:`

## Technologies Used

- [cloud-hypervisor](https://github.com/cloud-hypervisor/cloud-hypervisor): hypervisor that booting the machine
- [virtiofsd](https://gitlab.com/muhamad.azmy/virtiofsd/): used to share a host directory for the rootfs. we are using a forked version
- [rfs v1](https://github.com/threefoldtech/rfs/tree/v1): mounts the flist file into a directory serving as the lower layer of the overlay file system.
- `overlayfs`: mounts a read-write layer on the rootfs

## Script Walkthrough

- **Validation:**
  - Ensures necessary dependencies are available; downloads them if not.
  - Fails if no image is provided (either flist or directory).
- **Prepare Rootfs:**
  - If the image is a flist file, mounts it with `rfs`.
  - Creates a read-write layer with overlayfs mounted at `/tmp/overlay`.
  - Shares the overlay directory with `virtiofsd`.
  - For container images, mounts the cloud-container flist and adds it as the lower layer for overlayfs.
- **Prepare Boot:**
  - Specify paths for kernel/initrd/init script, or use defaults:
    - For machine images: `/boot/vmlinuz`, `/boot/initrd.img`, and `/sbin/init`.
    - For container images: `/kernel`, `/initramfs-linux.img`, and `/sbin/zinit`.
- **Prepare CIData:**
  - Uses provided image if `--cidata` flag is used.
  - Creates a basic cloud-init image with default config, which can be overridden with `--user`, `--pass`, `--name` flags.
- **Boot with Cloud-Hypervisor**
- **Cleanup:**
  - Kills all attached processes.
  - Unmounts and clears directories.

## Install dependencies

- **cloud-hypervisor**

    ```bash
    git clone https://github.com/cloud-hypervisor/cloud-hypervisor.git
    cd cloud-hypervisor
    cargo build --release
    sudo setcap cap_net_admin+ep ./target/release/cloud-hypervisor

    sudo ln -s $(realpath ./target/release/cloud-hypervisor) /usr/local/bin/cloud-hypervisor
    ```

- **virtiofsd**

    ```bash
    git clone https://gitlab.com/muhamad.azmy/virtiofsd.git
    cd virtiofsd
    cargo build --release
    sudo setcap cap_net_admin+ep ./target/release/virtiofsd

    sudo ln -s $(realpath ./target/release/virtiofsd) /usr/local/bin/virtiofsd
    ```

- **rfs**

    ```bash
    wget https://github.com/threefoldtech/rfs/releases/download/v1.1.1/rfs
    chmod +x ./rfs

    sudo ln -s $(realpath ./rfs) /usr/local/bin/rfs1
    ```

- **mkdosfs**
    only needed if the script gonna create the cidata image.

    ```bash
    apt-get install dosfstools
    ```

- **screen**

    ```bash
    apt-get install screen
    ```
