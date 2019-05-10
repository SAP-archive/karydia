{{ define "create-karydia-tls-secret.sh.tpl" }}
#!/bin/bash

# Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
# This file is licensed under the Apache Software License, v. 2 except as
# noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

set -euo pipefail

readonly secret_name="${SECRET_NAME:-karydia-tls}"
readonly cert_path="${CERT_PATH:-karydia.pem}"
readonly key_path="${KEY_PATH:-karydia-key.pem}"
readonly namespace="${NAMESPACE:-kube-system}"

kubectl create secret generic "${secret_name}" \
  --from-file=key.pem="${key_path}" \
  --from-file=cert.pem="${cert_path}" \
  --dry-run -o yaml |
  kubectl -n "${namespace}" apply -f -
{{ end }}
