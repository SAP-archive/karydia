# karydia features

| Feature | CLI flags | Control with Kubernetes resources | Status |
|---------|-----------|-----------------------------------|--------|
| Default Network Policy ([demo](demos/network/network.md))| `--enable-default-network-policy` <br/> `--default-network-policy-configmap` <br/> `--default-network-policy-excludes` | ConfigMap in `kube-system` namespace | Implemented but no reconciliation loop |
| Open Policy Agent | `--enable-opa-admission` | None | Implemented |
| Karydia Admission <br/> - seccomp ([demo](demos/seccomp/seccomp.md)) <br/> - service account token automount ([demo](demos/automount-service-account-token/automount-service-account-token.md)) | `--enable-karydia-admission` | Annotations on namespaces | Implemented |

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
of a secure-by-default cluster setup.

The features currently supported are:
1. Secure-by-default mouting of service account tokens
    - `change-default` sets `automountServiceAccountToken` of default ServiceAccounts to `false` when undefined
    - `change-all` sets `automountServiceAccountToken` of all ServiceAccounts to `false` when undefined
2. Enforcing a seccomp policy

It is configured with the following namespace annotations:

| Name | Type | Possible values |
|---|---|---|
|karydia.gardener.cloud/automountServiceAccountToken|string|`change-default` \| `change-all` 
|karydia.gardener.cloud/seccompProfile|string|Name of a valid profile, e.g. `runtime/default` or `localhost/my-profile`|

