# karydia features

| Feature | CLI flags | install/charts/values.yaml keys | Control with Kubernetes resources | Status |
|---------|-----------|---------------------------|-----------------------------------|--------|
| Karydia Config | `--config` | `config.name` | cluster-wide `KarydiaConfig` custom resource | Implemented |
| Default Network Policy | `--enable-default-network-policy` <br/> `--default-network-policy-excludes` | `feature.defaultNetworkPolicy` <br/> `config.networkPolicy` <br/> `config.defaultNetworkPolicyExcludes` | ConfigMap in `kube-system` namespace | Implemented but no reconciliation loop |
| Karydia Admission <br/> - seccomp ([demo](demos/seccomp.md)) <br/> - service account token automount | `--enable-karydia-admission` | `feature.karydiaAdmission` <br/> `config.seccompProfile` <br/> `config.automountServiceAccountToken` | Annotations on namespaces | Implemented |

## Karydia Config

With `--config` the default name of the cluster-wide `KarydiaConfig` custom resource, which karydia is using as default configuration, can be changed. By default, it is set to `karydia-config` which is the same as mentioned at `install/charts/values.yaml`:
```
config:
  name: "karydia-config"
```
If you want to adjust the default karydia behavior you can just modify the specific values at `install/charts/values.yaml` and re-deploy these changes to your karydia-controlled cluster:
```
helm upgrade karydia ./install/charts
```

## Default NetworkPolicy

When `--enable-default-network-policy` is set, karydia takes the network policy
found at deployed custom resource yaml `install/charts/templates/config.yaml` with key `networkPolicy` and installs it into all namespaces.

Particular namespaces can be excluded with `--default-network-policy-excludes`.

For easy change, adjust `enableDefaultNetworkPolicy` and `defaultNetworkPolicyExcludes` in `install/charts/values.yaml`.

Please note: an update of `networkPolicy` at `install/charts/values.yaml` does not update
previously deployed network policies. New namespaces created while karydia was
not running will not be updated when karydia starts.

The network policy is expected to be found under `.data.policy` in the configmap.

## Karydia Admission

Karydia Admission (`--enable-karydia-admission`) offers features with the goal of a secure-by-default cluster setup. You can enable/disable and configure this feature in the `install/charts/values.yaml` file.

The features currently supported are:
1. Secure-by-default mounting of service account tokens
    - `change-default` sets `automountServiceAccountToken` of default ServiceAccounts to `false` when undefined
    - `change-all` sets `automountServiceAccountToken` of all ServiceAccounts to `false` when undefined
2. Secure-by-default Seccomp profiles
    - Applies the given Seccomp profile to all pods that do not explicitly specify another profile.

It is configured with the following namespace annotations:

| Name | Type | Possible values |
|---|---|---|
|karydia.gardener.cloud/automountServiceAccountToken|string|`change-default` \| `change-all` 
|karydia.gardener.cloud/seccompProfile|string|Name of a valid profile, e.g. `runtime/default` or `localhost/my-profile`|

Karydia annotates the mutated resources with the at the time and context valid security settings:

| Resource | Annotation | Possible values |
|---|---|---|
| Pod |karydia.gardener.cloud/seccompProfile.internal | (`config` \| `namespace`) /(\<`profile-name`\>)
| ServiceAccount | karydia.gardener.cloud/automountServiceAccountToken.internal | (`config` \| `namespace`) /(`change-default` \| `change-all`)|

### karydia.gardener.cloud/automountServiceAccountToken

The feature defaults a service account's `automountServiceAccountToken` to false in cases 5, 6 and 7 of the following table. With setting `change-default` this is enforced for default service accounts, with setting `change-all` this is enforced for all service accounts (apart the ones in the `kube-system` namespace). The actual behavior of auto-mounting only changes in case 5, when `automountServiceAccountToken` is also undefined in the Pod definition. 

| # | service account | pod | k8s behavior | karydia behavior |
|---|-----------------|-----|--------------|-----------------|
|1| true | true | true | true |
|2| false | true | true | true |
|3| true | false | false | false |
|4| false | false | false | false |
|5| **not defined** | not defined | true | **false** |
|6| **not defined** | true | true | true |
|7| **not defined** | false | false | false |
|8| true | not defined | true | true |
|9| false | not defined | false | false |
