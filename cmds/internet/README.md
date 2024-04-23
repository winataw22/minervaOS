# Internet

The internet module is responsible to connetct zos node to the internet.

## How it works

The internet module bootstraps the private zos network as follows:

- Find a physical interface that can get an IPv4 over DHCP or use `priv vlan` if configured as kernel param.

- Create a bridge called `zos` and attach the interface to it.

- Start a DHCP daemon after the Bridge and interface are brought UP to get an IP.

- Test the internet connetction by trying to connect to some addresses `"bootstrap.grid.tf:http", "hub.grid.tf:http"`

## Build

The internet binary is build as a part of the build process of zos base image as follows:

- The `0-initramfs` first installs bootstrap.

- The installation builds and copys the internet binary to initramfs root bin along with other bootstrap binaries.
