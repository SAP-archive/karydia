#!/bin/bash

set -ex

go env
go get -d -t -v ./...
export CGO_LDFLAGS="-L/usr/lib -lopenblas"
go test -a -v ./...
if [[ $TRAVIS_SECURE_ENV_VARS = "true" ]]; then bash -c "$GOPATH/src/gonum.org/v1/netlib/.travis/test-coverage.sh"; fi
