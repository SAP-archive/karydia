# karydia features

| Feature | CLI flags | install/charts/values.yaml keys | Control with Kubernetes resources | Status |
|---------|-----------|---------------------------|-----------------------------------|--------|
| Karydia Config | `--config` | `config.name` | cluster-wide `KarydiaConfig` custom resource | Implemented |
| Karydia Network Policy | `--enable-default-network-policy` <br/> `--default-network-policy-excludes` | `features.defaultNetworkPolicy` <br/> `config.networkPolicy` <br/> `config.defaultNetworkPolicyExcludes` | cluster-wide `KarydiaNetworkPolicy` custom resource | Implemented |
| Karydia Admission <br/> - seccomp ([demo](demos/seccomp.md)) <br/> - service account token automount | `--enable-karydia-admission` | `features.karydiaAdmission` <br/> `config.seccompProfile` <br/> `config.automountServiceAccountToken` | Annotations on namespaces | Implemented |

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

## Karydia Network Policy

When `--enable-network-policy` is set, karydia takes the custom karydia network policy resource
found at deployed custom resource yaml `install/charts/templates/config.yaml` with key `networkPolicy` as a template for a network policy, which will be installed into all namespaces.

Particular namespaces can be excluded with `--default-network-policy-excludes`.

For easy change, adjust `enableDefaultNetworkPolicy` and `defaultNetworkPolicyExcludes` in `install/charts/values.yaml`. You can enable/disable this feature by setting `defaultNetworkPolicy` to true`/`false`.

You can configure the default network policy for a specific namespace with the following namespace annotation:


| Name | Type | Possible values |
|---|---|---|
|"karydia.gardener.cloud/networkPolicy"|string|Name of a deployed karydia network policy, e.g. `karydia-default-network-policy-l2`|

Please note: an update of `networkPolicy` at `install/charts/values.yaml` does not update
previously deployed network policies. New namespaces created while karydia was
not running will not be updated when karydia starts.


The current network policy called `karydia-default-network-policy` has two security measures:
1. block access to host network (AWS only)
2. block access to Meta Data Services (AWS, GCP, Azure, Alibaba Cloud)

Note: The network policy is still quite open. It uses a blacklisting approach and does not block Internet access (Egress).

The `network-policy-l2` blocks the Egress ensuring that no pod can reach the Internet.

The `network-policy-l3` blocks the Egress as well as allows communication only within a namespace.

Karydia annotates the created network policy resources with the at the time and context valid security settings:

| Resource | Annotation | Possible values |
|---|---|---|
| NetworkPolicy |karydia.gardener.cloud/networkPolicy.internal | (`config` \| `namespace`) /(\<`network-policy-name`\>) |

## Karydia Admission

Karydia Admission (`--enable-karydia-admission`) offers features with the goal of a secure-by-default cluster setup. You can enable/disable this feature by setting `karydiaAdmission` to `true`/`false`.

The features currently supported are:
1. Secure-by-default mounting of service account tokens
    - `change-default` sets `automountServiceAccountToken` of default ServiceAccounts to `false` when undefined
    - `change-all` sets `automountServiceAccountToken` of all ServiceAccounts to `false` when undefined
    - `no-change`represents the fallback option and uses the default Kubernetes setting (e.g. sets `automountServiceAccountToken` of ServiceAccounts to `true`)
2. Secure-by-default Seccomp profiles
    - Applies the given Seccomp profile to all pods that do not explicitly specify another profile.
    - Place your custom profiles into `install/charts/custom-seccomp-profiles/` and karydia distribute and manage them over all nodes in your cluster.
    - `unconfined` represents the fallback option and will not apply any Seccomp profile to any pod.
3. Secure-by-default User and Group context for pods
    - `nobody` set the user and group of all pods that do not explicitly specify another security context to id `65534`.
    - `none` represents the fallback option and disables the feature.

It is configured with the following namespace annotations:

| Name | Type | Possible values |
|---|---|---|
|karydia.gardener.cloud/automountServiceAccountToken|string|`change-default` \| `change-all` \| `no-change`|
|karydia.gardener.cloud/podSecurityContext|string|`nobody` \| `none`|
|karydia.gardener.cloud/seccompProfile|string| `runtime/default` \| `localhost/my-profile` \| `unconfined`|

Karydia annotates the mutated resources with the at the time and context valid security settings:

| Resource | Annotation | Possible values |
|---|---|---|
| Pod |karydia.gardener.cloud/seccompProfile.internal | (`config` \| `namespace`) /(\<`profile-name`\>) |
| Pod |karydia.gardener.cloud/podSecurityContext.internal | (`config` \| `namespace`) /(`nobody` \| `none`) |
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

## Karydia Exclusion Handling

Namespaces and other objects can be opted out of being "watched" by karydia. Therefore, there are two options:
- (nearly) each feature provides its own annotation for namespaces and/or other objects to be ignored by the respective karydia feature - see feature descriptions above
- [values.yaml](../install/charts/values.yaml), which provides karydia (component) installation configurations, provides two blocks called `exclusionNamespaceLabels` and `exclusionObjectLabels`. These blocks define either namespace or other object labels. If they are matched by either namespace or object the karydia webhooks filter them out and, thus, they get fully excluded/ignored by karydia. These settings need to be adjusted before running the installation of karydia.

:warning: Karydia's network policy feature works differently, without the use of webhooks and, hence, this feature is independent from that configuration setting.
