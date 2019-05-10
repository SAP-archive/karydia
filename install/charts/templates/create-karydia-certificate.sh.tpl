{{ define "create-karydia-certificate.sh.tpl" }}
#!/bin/bash

# Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
# This file is licensed under the Apache Software License, v. 2 except as
# noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

set -euo pipefail

for req in kubectl openssl; do
  if ! command -v "${req}" &>/dev/null; then
    echo "'${req}' required but not found" >&2
    exit 1
  fi
done


readonly service_name="${SERVICE_NAME:-karydia}"
readonly secret_name="${SECRET_NAME:-karydia-tls}"
readonly cert_path="${CERT_PATH:-karydia.pem}"
readonly key_path="${KEY_PATH:-karydia-key.pem}"
readonly namespace="${NAMESPACE:-kube-system}"

readonly csr_name="${service_name}.${namespace}"

if [[ -e "${cert_path}" ]] || [[ -e "${key_path}" ]]; then
  echo "ERROR: found existing '${cert_path}' or '${key_path} - aborting" >&2
  exit 1
fi

readonly tmp_dir="$(mktemp -d /tmp/karydia-csr-XXXX)"

trap 'rm -rf "${tmp_dir}"' EXIT

cat <<EOF >"${tmp_dir}/csr.conf"
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${service_name}
DNS.2 = ${service_name}.${namespace}
DNS.3 = ${service_name}.${namespace}.svc
DNS.4 = 127.0.0.1
DNS.5 = localhost
EOF

openssl genrsa -out "${key_path}" 2048
openssl req -new \
  -key "${key_path}" \
  -subj "/CN=${service_name}.${namespace}.svc" \
  -config "${tmp_dir}/csr.conf" \
  -out "${tmp_dir}/karydia.csr"

# Clean-up any previously created CSR for our service
kubectl delete csr "${csr_name}" &>/dev/null || true

cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: ${csr_name}
spec:
  groups:
  - system:authenticated
  request: '$(base64 "${tmp_dir}/karydia.csr" | tr -d '\r\n')'
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

echo "Waiting for CSR '${csr_name}' to be created ..."
until kubectl get csr "${csr_name}" &>/dev/null; do sleep 1; done

# Approve the CSR
kubectl certificate approve "${csr_name}"

# Verify certificate has been signed
for _ in {1..10}; do
  cert="$(kubectl get csr "${csr_name}" -o jsonpath='{.status.certificate}')"
  if [[ -n "${cert}" ]]; then
    break
  fi
  sleep 1
done

if [[ -z "${cert}" ]]; then
  echo "ERROR: after approving CSR '${csr_name}', the signed certificate did not appear on the resource - aborting" >&2
  exit 1
fi

echo "${cert}" | openssl base64 -d -A -out "${cert_path}"
{{ end }}
