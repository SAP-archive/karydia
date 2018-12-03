# karydia - Kubernetes security walnut

Status: alpha, work in progress

## Getting started

### Setup development and test environment

First, install [minikube](https://kubernetes.io/docs/setup/minikube/).

Then the following commands can be used to install karydia as a mutating
webhook plus a default `KarydiaSecurityPolicy`:

```
minikube start --kubernetes-version v1.12.3

rm karydia*pem
./scripts/create-karydia-certificate
./scripts/create-karydia-tls-secret

eval $(minikube docker-env)

make container

kubectl apply -f manifests/karydia.yml

kubectl create configmap -n kube-system karydia-default-network-policy --from-literal policy="$(<manifests/example-default-network-policy.yml)"

kubectl apply -f manifests/crd-karydia-security-policy.yml

./scripts/configure-karydia-webhook
```

Take a quick look at the logs to see if karydia started:

```
kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f -c karydia

time="2018-11-09T10:47:50Z" level=info msg="Listening on 0.0.0.0:33333"
[...]
```

### Demo

Please see [docs/karydia-demo.md](docs/karydia-demo.md) for a demo of some karydia features.

## Configuration options

### karydia admission

If enabled (`--enable-karydia-admission`), karydia acts as an admission plugin
to enforce certain settings with the goal of hardening a cluster setup.

Currently, the following settings are available to be set per namespace
as namespace annotations:

| Name | Type | Possible values |
|---|---|---|
|karydia.gardener.cloud/automountServiceAccountToken|string|"forbidden" or "non-default"|
|karydia.gardener.cloud/seccompProfile|string|Name of a valid profile, e.g. "runtime/default" or "localhost/my-profile"|

### KarydiaSecurityPolicy

```
apiVersion: karydia.gardener.cloud/v1alpha1
kind: KarydiaSecurityPolicy
metadata:
  name: example
spec:
  pod:
    # automountServiceAccountToken can be 'forbidden' or 'non-default'
    automountServiceAccountToken: "forbidden"
    # seccompProfile can be any valid seccomp profile
    # If the pod does not define a profile, it will be patched
    # If the pod defines a different profile, it will be rejected
    seccompProfile: "docker/default"
```

### Default NetworkPolicy

When `--enable-default-network-policy` is set, karydia takes the network policy
found at `--default-network-policy-configmap` and installs it into all namespaces.

Particular namespaces can be excluded with `--default-network-policy-excludes`.

Please note: an update to `--default-network-policy-configmap` does not update
previously deployed network policies.

The network policy is expected to be found under `.data.policy` in the
configmap.
