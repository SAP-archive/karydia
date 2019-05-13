{{ define "configure-karydia-webhook.sh.tpl" }}
#!/bin/bash

# Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
# This file is licensed under the Apache Software License, v. 2 except as
# noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

set -euo pipefail

ca_bundle="$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\r\n')"
if [[ -z "${ca_bundle}" ]]; then
  echo "ERROR: extension-apiserver-authentication config map with CA bundle not found - aborting" >&2
  exit 1
fi

cat <<EOF | sed -e "s|§CA_BUNDLE§|${ca_bundle}|g" | kubectl apply -f -
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{ .Values.metadata.name }}-webhook
  labels:
    app: {{ .Values.metadata.labelApp }}
webhooks:
  - name: {{ .Values.metadata.apiGroup }}
    failurePolicy: Ignore
    clientConfig:
      service:
        name: {{ .Values.metadata.name }}
        namespace: {{ .Values.metadata.namespace }}
        path: "/webhook/validating"
      caBundle: §CA_BUNDLE§
    rules:
      # https://github.com/kubernetes/kubernetes/blob/v1.12.1/staging/src/k8s.io/api/admissionregistration/v1beta1/types.go
      - operations:
        - CREATE
        - UPDATE
        apiGroups: ["*"]
        apiVersions: ["*"]
        resources:
        - nodes
        - namespaces
        - pods
        - pods/status
        - endpoints
        - persistentvolumes
        - validatingwebhookconfigurations
        - mutatingwebhookconfigurations
EOF

cat <<EOF | sed -e "s|§CA_BUNDLE§|${ca_bundle}|g" | kubectl apply -f -
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: {{ .Values.metadata.name }}-webhook
  labels:
    app: {{ .Values.metadata.labelApp }}
webhooks:
  - name: {{ .Values.metadata.apiGroup }}
    failurePolicy: Ignore
    clientConfig:
      service:
        name: {{ .Values.metadata.name }}
        namespace: {{ .Values.metadata.namespace }}
        path: "/webhook/mutating"
      caBundle: §CA_BUNDLE§
    rules:
      # https://github.com/kubernetes/kubernetes/blob/v1.12.1/staging/src/k8s.io/api/admissionregistration/v1beta1/types.go
      - operations:
        - CREATE
        - UPDATE
        apiGroups: ["*"]
        apiVersions: ["*"]
        resources:
        - nodes
        - namespaces
        - pods
        - pods/status
        - endpoints
        - persistentvolumes
        - validatingwebhookconfigurations
        - mutatingwebhookconfigurations
EOF
{{ end }}
