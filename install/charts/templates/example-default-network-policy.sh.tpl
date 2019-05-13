{{ define "example-default-network-policy.sh.tpl" }}
#!/bin/bash

# Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
# This file is licensed under the Apache Software License, v. 2 except as
# noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

kubectl delete configmap -n {{ .Values.metadata.namespace }} {{ .Values.metadata.name }}-default-network-policy 2> /dev/null

networkPolicy=$(cat <<EOF
apiVersion: extensions/v1beta1
kind: NetworkPolicy
metadata:
  labels:
    app: {{ .Values.metadata.labelApp }}
  name: {{ .Values.metadata.name }}-default-network-policy
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Egress
EOF
)

kubectl create configmap -n {{ .Values.metadata.namespace }} {{ .Values.metadata.name }}-default-network-policy --from-literal policy="${networkPolicy}"

{{ end }}
