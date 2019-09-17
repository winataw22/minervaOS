#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list ./... | grep -Ev "stubs|flist|provision|network|storage|gedis" ); do
    echo "test $d"
    go test -vet=off -coverprofile=profile.out -race -covermode=atomic "$d"
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done