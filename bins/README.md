# Extra Binaries

Zero-OS image comes with minimal software suite shipped and some software required
on runtime are downloaded via flist later, when the system booted.

This directory contains a build script suite to build theses runtime dependencies.

## Requirement

The whole build system is intended to be used inside a Docker container using `ubuntu:18.04` image
like the initramfs build script. This can be used inside Github Actions Steps.

## Package system

The build system use a simple mechanism which build differents « packages ». Each packages
are defined on the package directory. A package is a directory, the directory name is the package name.
Inside this directory, you need a bash script called with the package name.

Eg: `packages/mysoftware/mysoftware.sh`

You can get an extra directory next to your bash script, called `files` and put inside some files
you need during build phase.

To build your package, you can invoke the build script like this: `bash bins-extra.sh --package mysoftware`.

The build script takes care to set up an environment easy to use inside your package script.
You have differents variables setup you can use:
- `FILESDIR`: the path where your files next to your build script are located
- `WORKDIR`: a temporary directory you can use to build your software
- `ROOTDIR`: root directory you can use to put your final files to packages
- `DISTDIR`: a temporary directory where you can download stuff (like source code, etc.)

Your build script have to get a function called `build_${pkgname}` eg: `build_mysoftware`.
This function have to deal with everything you want and put final files into `ROOTDIR`. When this
is done, the build script will ensure all the shared libraries needed by your binaries will be
available on `ROOTDIR`.

Take a look on existing build script to see how it works. Script won't look foreign for you if you know
a how ebuild scripts works for Gentoo :)
