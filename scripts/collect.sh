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
for sub in $(bin/zos --list); do
    ln -s zos ${archive}/bin/${sub}
done
cp -r etc/* ${archive}/etc/
