# Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

# build
FROM golang:1.12.0 as build-stage
RUN mkdir -p /go/src/github.com/karydia/karydia/
WORKDIR /go/src/github.com/karydia/karydia/
COPY ./ ./
RUN make
RUN make test

# dev-image (only for development)
FROM k8s.gcr.io/alpine-with-bash:1.0 as dev-image
RUN apk update && apk add inotify-tools
COPY --from=build-stage /go/src/github.com/karydia/karydia/bin/karydia /usr/local/bin/karydia
COPY ./scripts/hotswap-dev /usr/local/bin/hotswap-dev

# prod-image (production usage)
FROM k8s.gcr.io/debian-base-amd64:0.4.0 as prod-image
COPY --from=build-stage /go/src/github.com/karydia/karydia/bin/karydia /usr/local/bin/karydia
USER 65534:65534

