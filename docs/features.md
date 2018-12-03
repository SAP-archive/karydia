# karydia features

| Feature | CLI flags | Control with Kubernetes resources | Status |
|---------|-----------|-----------------------------------|--------|
| Default Network Policy | `--enable-default-network-policy` <br/> `--default-network-policy-configmap` <br/> `--default-network-policy-excludes` | ConfigMap in `kube-system` namespace | Implemented but no reconciliation loop |
| Open Policy Agent | `--enable-opa-admission` | None | Implemented |
| Karydia Admission <br/> - seccomp <br/> - service account token automount | `--enable-karydia-admission` | Annotations on namespaces | Implemented |

## Default NetworkPolicy

When `--enable-default-network-policy` is set, karydia takes the network policy
found at `--default-network-policy-configmap` and installs it into all namespaces.

Particular namespaces can be excluded with `--default-network-policy-excludes`.

Please note: an update to `--default-network-policy-configmap` does not update
previously deployed network policies. New namespaces created while karydia was
not running will not be updated when karydia starts.

The network policy is expected to be found under `.data.policy` in the
configmap.

## Open Policy Agent

When `--enable-opa-admission` is set, karydia will implement an admission
controller that forwards the request to an [Open Policy
Agent](https://www.openpolicyagent.org/) endpoint specified by
`--opa-api-endpoint`.

Complex policies can be loaded in OPA using the rego language without needing
additional code in Golang.

## Karydia admission

Karydia Admission (`--enable-karydia-admission`) offers features with the goal
of hardening a cluster setup.

The features currently supported are:
- prevent service account token automounts
- enforcing a seccomp policy

It is configured with the following namespace annotations:

| Name | Type | Possible values |
|---|---|---|
|karydia.gardener.cloud/automountServiceAccountToken|string|"forbidden" or "non-default"|
|karydia.gardener.cloud/seccompProfile|string|Name of a valid profile, e.g. "runtime/default" or "localhost/my-profile"|

