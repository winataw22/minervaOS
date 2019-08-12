#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list ./... | grep -v stubs | grep -v flist | grep -v provision | grep -v network | grep -v storage ); do
    echo "test $d"
    go test -vet=off -coverprofile=profile.out -covermode=atomic "$d"
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done