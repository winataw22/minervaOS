# The package

this package include miscellaneous files that don't really fit a certain package

## Files

### `prop.script`

this file is a **HARD LINK** to the file with the same name at `/bootstrap/usr/share/udhcp/` this file exists in two places
because

- It needs to be part of the 0-initramfs (zos kernel) because it's needed to exist before the node has internet connection
and the bootstrap directory is what is included during the kernel build
- But it also need to be in a package because that any node that is booted with an older build of the kernel must also get the same
file

I am not sure if the hard-link is maintained in a github repo (I assume not) so any changes to the prop.script must be maintained in
the two copies
