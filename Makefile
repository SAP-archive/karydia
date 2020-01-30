# Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
# This file is licensed under the Apache Software License, v. 2 except as
# noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DEV_POSTFIX=-dev
PROD_IMAGE=eu.gcr.io/gardener-project/karydia/karydia
DEV_IMAGE=$(PROD_IMAGE)$(DEV_POSTFIX)

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

.PHONY: build-debug
build-debug:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/karydia \
		-gcflags="all=-N -l" \
		-ldflags "-X github.com/karydia/karydia.Version=$(VERSION)" \
		cli/main.go

.PHONY: container
container:
	docker build --target prod-image -t $(PROD_IMAGE) .

.PHONY: container-dev
container-dev:
	docker build --target dev-image -t $(DEV_IMAGE) .

.PHONY: deploy-dev
deploy-dev:
	for i in $(shell kubectl get pods -n=karydia --selector=app=karydia --output=jsonpath='{.items[*].metadata.name}'); do \
		kubectl cp bin/karydia -n karydia $$i:/usr/local/bin/karydia-dev; \
	done

.PHONY: debug-dev
debug-dev:
	kubectl port-forward -n karydia pod/$(shell kubectl get pods -n=karydia --selector=app=karydia --output=jsonpath='{.items[0].metadata.name}') 2345

.PHONY: codegen
codegen:
	hack/update-codegen.sh

.PHONY: test-only
test-only:
	go test $(shell go list ./... | grep -v /vendor/ | grep -v /tests/)

.PHONY: fmt
fmt:
	@if [ -n "$(shell gofmt -l pkg cli tests)" ]; then \
    		echo "Go code is not formatted!";\
    		exit 1;\
	fi

.PHONY: test
test: fmt test-only

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=cov.out $(shell go list ./... | grep -v /vendor/ | grep -v /tests/)
	go tool cover -html=cov.out

.PHONY: e2e-test
e2e-test:
	go test -v ./tests/e2e/... --server $(KUBERNETES_SERVER) --kubeconfig $(KUBECONFIG_PATH)
