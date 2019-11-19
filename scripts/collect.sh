#!/bin/sh
set -e

# This file is used by CI to build an archive with
# all the binaries, and config files for flist building

archive=$1

if [ -z "${archive}" ]; then
    echo "missing argument" >&2
    exit 1
fi

mkdir -p ${archive}/bin ${archive}/etc
cp bin/* ${archive}/bin/
cp -r etc/* ${archive}/etc/