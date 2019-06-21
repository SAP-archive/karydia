{{ define "example-default-network-policy.sh.tpl" }}
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
