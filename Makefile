IMAGE_REPO=karydia
IMAGE_NAME=karydia

VERSION=$(shell git describe --tags --always --dirty)

.PHONY: all
all: build

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/karydia \
		-ldflags "-s -X github.com/kinvolk/karydia/cli/cmd.version=$(VERSION)" \
		cli/main.go

.PHONY: container
container: build
	docker build -t $(IMAGE_REPO)/$(IMAGE_NAME) .

.PHONY: codegen
codegen:
	hack/update-codegen.sh

.PHONY: test
test:
	go test ./...
