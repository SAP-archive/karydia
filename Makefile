VERSION=$(shell git describe --tags --always --dirty)

.PHONY: all
all: build

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/karydia \
		-ldflags "-s -X github.com/kinvolk/karydia/cli/cmd.version=$(VERSION)" \
		cli/main.go
