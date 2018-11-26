# karydia - Kubernetes security walnut

Status: alpha, work in progress

## Getting started

### Setup development and test environment

First, install [minikube](https://kubernetes.io/docs/setup/minikube/).

Then the following commands can be used to install karydia as a mutating
webhook plus a default `KarydiaSecurityPolicy`:

```
minikube start --kubernetes-version v1.12.2

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
E1109 10:47:50.413218       1 reflector.go:134] github.com/kinvolk/karydia/pkg/client/informers/externalversions/factory.go:117: Failed to list *v1alpha1.KarydiaSecurityPolicy: the server could not find the requested resource (get karydiasecuritypolicies.karydia.gardener.cloud)
E1109 10:47:51.440790       1 reflector.go:134] github.com/kinvolk/karydia/pkg/client/informers/externalversions/factory.go:117: Failed to list *v1alpha1.KarydiaSecurityPolicy: the server could not find the requested resource (get karydiasecuritypolicies.karydia.gardener.cloud)
E1109 10:47:52.458124       1 reflector.go:134] github.com/kinvolk/karydia/pkg/client/informers/externalversions/factory.go:117: Failed to list *v1alpha1.KarydiaSecurityPolicy: the server could not find the requested resource (get karydiasecuritypolicies.karydia.gardener.cloud)
time="2018-11-09T10:47:56Z" level=info msg="Enqueuing \"default\""
time="2018-11-09T10:47:56Z" level=info msg="Successfully synced \"default\""
[...]
```

Since it can take a moment for the `KarydiaSecurityPolicy` CRD to be available,
you might see a few error messages as above.

### Demo

If no `KarydiaSecurityPolicy` was created yet or your current user is
not allowed to use any of the available policies, all requests will be
allowed by default.

```
$ kubectl run --rm -ti --restart=Never --image busybox busybox -- echo hello world
hello world
pod "busybox" deleted
```

Let's add an example policy:

```
kubectl apply -f manifests/example-karydia-security-policy.yml
```

Now the same `kubectl run` command should fail according to the policy:

```
$ kubectl run --rm -ti --restart=Never --image busybox busybox -- echo hello world
Error from server (InternalError): Internal error occurred: admission webhook "karydia.gardener.cloud" denied the request: map[example-restrictive:[automount of service account not allowed]]
```

Let's try to comply with the KSP:

```
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: busybox
spec:
  automountServiceAccountToken: false
  containers:
  - image: busybox
    imagePullPolicy: IfNotPresent
    name: busybox
    command: ["sleep", "99999"]
EOF
```

That worked:

```
$ kubectl get pods
NAME      READY     STATUS    RESTARTS   AGE
busybox   1/1       Running   0          26s
```

Also, according to the policy, karydia has not only verified
that `automountServiceAccountToken` is set to false but also modified the
pod to use the configured seccomp profile. You can verify that with:

```
$ kubectl get pods busybox -o jsonpath='{.metadata.annotations.seccomp\.security\.alpha\.kubernetes\.io/pod}'
```

## Configuration options

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
