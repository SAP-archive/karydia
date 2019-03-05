IMAGE_REPO=karydia
IMAGE_NAME=karydia

KUBERNETES_SERVER ?= ""
KUBECONFIG_PATH ?= "$(HOME)/.kube/config"

VERSION=$(shell git describe --tags --always --dirty)

.PHONY: all
all: build

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/karydia \
		-ldflags "-s -X github.com/karydia/karydia.Version=$(VERSION)" \
		cli/main.go

.PHONY: container
container: build
	docker build -t $(IMAGE_REPO)/$(IMAGE_NAME) .

.PHONY: codegen
codegen:
	hack/update-codegen.sh

.PHONY: test
test:
	go test $(shell go list ./... | grep -v /vendor/ | grep -v /tests/)

.PHONY: e2e-test
e2e-test:
	go test -v ./tests/e2e/... --server $(KUBERNETES_SERVER) --kubeconfig $(KUBECONFIG_PATH)
