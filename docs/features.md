# Karydia features

| Feature | CLI flags | install/charts/values.yaml keys | Control with Kubernetes resources | Status |
|---------|-----------|---------------------------|-----------------------------------|--------|
| Karydia Config | `--config` | `config.name` | cluster-wide `KarydiaConfig` custom resource | Implemented |
| Karydia Network Policy | `--enable-default-network-policy` <br/> `--default-network-policy-excludes` | `features.defaultNetworkPolicy` <br/> `config.networkPolicies` <br/> `config.defaultNetworkPolicyExcludes` | cluster-wide `KarydiaNetworkPolicy` custom resource | Implemented |
| Karydia Admission <br/> - seccomp ([demo](demos/seccomp.md)) <br/> - service account token automount | `--enable-karydia-admission` | `features.karydiaAdmission` <br/> `config.seccompProfile` <br/> `config.automountServiceAccountToken` | Annotations on namespaces | Implemented |

## Karydia Config

With `--config` the default name of the cluster-wide `KarydiaConfig` custom resource, which Karydia is using as default configuration, can be changed. By default, it is set to `karydia-config` which is the same as mentioned at `install/charts/values.yaml`:
```
config:
  name: "karydia-config"
```
If you want to adjust the default Karydia behavior you can just modify the specific values at `install/charts/values.yaml` and re-deploy these changes to your Karydia-controlled cluster:
```
helm upgrade karydia ./install/charts
```

## Karydia Network Policy

When `--enable-network-policy` is set, Karydia takes the custom Karydia network policy resources
found at the deployed custom resource yaml `install/charts/templates/config.yaml` with key `networkPolicies` as a template for a network policy, which will be installed into all namespaces. You can define one or multiple default network policies using a `;`-separated syntax (e.g. `karydia-default-network-policy-l2;karydia-default-network-policy-l3`).

Particular namespaces can be excluded with `--default-network-policy-excludes`.

For easy change, adjust `networkPolicies` and `defaultNetworkPolicyExcludes` in `install/charts/values.yaml`. You can enable/disable this feature by setting `defaultNetworkPolicy` to `true`/`false`.

You can configure the default network policies for a specific namespace with the following namespace annotation:

| Name | Type | Possible values |
|---|---|---|
|"karydia.gardener.cloud/networkPolicy"|string|Name of a deployed Karydia network policy, e.g. `karydia-default-network-policy-l2;karydia-default-network-policy-l3`|

Please note: an update of `networkPolicies` at `install/charts/values.yaml` does not update previously deployed network policies. New namespaces created while Karydia was not running will not be updated when Karydia starts.

Karydia provides three different levels of network policies:
1. `karydia-default-network-policy-l1`: Block metadata service, access to kube-system namespace and access to the host network.
2. `karydia-default-network-policy-l2`: Level 1 security implications and block internet access (egress).
3. `karydia-default-network-policy-l3`: Level 2 security implications and block communication between different namespaces.

You can specify one or multiple default network policies (including custom policies). By default, `karydia-default-network-policy-l1` is deployed in every namespace. You can set the cloud provider in the `install/charts/values.yaml` (key `cloudProvider`) to `Azure`, `AWS`, `GCP`, `OpenStack` or `AliCloud`. If you cannot specify the cloud provider, use the backup empty value (i.e. `""`).

Karydia annotates the created network policy resources with the currently valid security settings (depending on time and context):

| Resource | Annotation | Possible values |
|---|---|---|
| NetworkPolicy |karydia.gardener.cloud/networkPolicy.internal | (`config` \| `namespace`) /(\<`network-policy-name`\>) |

## Karydia Admission

Karydia Admission (`--enable-karydia-admission`) offers features with the goal of a secure-by-default cluster setup. You can enable/disable this feature by setting `karydiaAdmission` to `true`/`false`.

The currently supported features are:
1. Secure-by-default mounting of service account tokens
    - `change-default` sets `automountServiceAccountToken` of default ServiceAccounts to `false` when undefined
    - `change-all` sets `automountServiceAccountToken` of all ServiceAccounts to `false` when undefined
    - `no-change`represents the fallback option and uses the default Kubernetes setting (e.g. sets `automountServiceAccountToken` of ServiceAccounts to `true`)
2. Secure-by-default Seccomp profiles
    - Applies the given Seccomp profile to all pods that do not explicitly specify another profile.
    - Place your custom profiles into `install/charts/custom-seccomp-profiles/` and Karydia distribute and manage them over all nodes in your cluster.
    - `unconfined` represents the fallback option and will not apply any Seccomp profile to any pod.
3. Secure-by-default User and Group context for pods
    - `nobody` set the user and group of all pods that do not explicitly specify another security context to id `65534`.
    - `none` represents the fallback option and disables the feature.
4. Secure-by-default security context for containers
    - `allowPrivilegeEscalation` is set to false if it is not explicitly specified.

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

### Karydia.gardener.cloud/automountServiceAccountToken

The feature defaults a service account's `automountServiceAccountToken` to false in cases 5, 6 and 7 of the following table. With setting `change-default` this is enforced for default service accounts, with setting `change-all` this is enforced for all service accounts (apart the ones in the `kube-system` namespace). The actual behavior of auto-mounting only changes in case 5, when `automountServiceAccountToken` is also undefined in the Pod definition. 

| # | service account | pod | k8s behavior | Karydia behavior |
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

Namespaces and other objects can be opted out of being "watched" by Karydia. Therefore, there are two options:
- (nearly) each feature provides its own annotation for namespaces and/or other objects to be ignored by the respective Karydia feature (see the feature descriptions above).
- [values.yaml](../install/charts/values.yaml), which provides Karydia (component) installation configurations, provides two blocks called `exclusionNamespaceLabels` and `exclusionObjectLabels`. These blocks define either namespace or other object labels. If they are matched by either a namespace or an object, the Karydia webhooks filter them out and, thus, they get fully excluded/ignored by Karydia. These settings need to be adjusted before running the installation of Karydia.

:warning: Karydia's network policy feature works differently, without the use of webhooks and, hence, this feature is independent from that configuration setting.
