{{ define "create-karydia-tls-secret.sh.tpl" }}
#!/bin/bash

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

set -euo pipefail

readonly secret_name="${SECRET_NAME:-karydia-tls}"
readonly cert_path="${CERT_PATH:-karydia.pem}"
readonly key_path="${KEY_PATH:-karydia-key.pem}"
readonly namespace="${NAMESPACE:-{{ .Values.metadata.namespace }}}"

kubectl create secret generic "${secret_name}" \
  --from-file=key.pem="${key_path}" \
  --from-file=cert.pem="${cert_path}" \
  --dry-run -o yaml |
  kubectl -n "${namespace}" apply -f -
{{ end }}
