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

kubectl apply -f manifests/crd-karydia-security-policy.yml

kubectl apply -f manifests/default-karydia-security-policy.yml

./scripts/configure-karydia-webhook
```

Take a quick look at the logs to see if karydia started:

```
kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f

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

With the default `KarydiaSecurityPolicy` applied, creating a regular pod
should fail:

```
kubectl run --rm -ti --restart=Never --image busybox busybox
Error from server (InternalError): Internal error occurred: admission webhook "karydia.gardener.cloud" denied the request: map[default:[[automount of service account 'default' not allowed]]]
```

We are not allowed to automount the default service account in a pod.

Let's try again with a distinct service account for the pod:

```
kubectl create sa testsa
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: busybox
spec:
  serviceAccountName: "testsa"
  containers:
  - image: busybox
    imagePullPolicy: IfNotPresent
    name: busybox
    command: ["sleep", "99999"]
EOF
pod/busybox created
```

That worked:

```
$ kubectl get pod busybox
NAME      READY     STATUS    RESTARTS   AGE
busybox   1/1       Running   0          44s
```

Let's create a new namespace and a user with [edit](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles)
role:

```
kubectl create ns jane
kubectl create sa -n jane default
kubectl create sa -n jane jane
kubectl create rolebinding -n jane jane-edit --clusterrole=edit --serviceaccount=jane:jane
```

Try to run a pod:

```
kubectl --as system:serviceaccount:jane:jane -n jane run --rm -ti --restart=Never --image busybox busybox
Error from server (InternalError): Internal error occurred: admission webhook "karydia.gardener.cloud" denied the request: no karydia security policy found to validate against
```

That didn't work. While the service account has permission to run a pod,
we have not allowed usage of any `KarydiaSecurityPolicy` yet and by default
all requests are denied.

Let's add a new KSP and allow service account `jane` to use it:

```
kubectl apply -f manifests/example-karydia-security-policy.yml
kubectl create rolebinding -n jane jane-ksp --clusterrole=ksp-example-restrictive --serviceaccount=jane:jane
```

Again, try to run a pod:

```
kubectl --as system:serviceaccount:jane:jane -n jane run --rm -ti --restart=Never --image busybox busybox
Error from server (InternalError): Internal error occurred: admission webhook "karydia.gardener.cloud" denied the request: map[example-restrictive:[automount of service account not allowed]]
```

The KSP doesn't allow mounting of a service account token. Let's try again
without service account token automount:

```
cat <<EOF | kubectl --as system:serviceaccount:jane:jane apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: busybox
  namespace: jane
spec:
  automountServiceAccountToken: false
  containers:
  - image: busybox
    imagePullPolicy: IfNotPresent
    name: busybox
    command: ["sleep", "99999"]
EOF
pod/busybox created
```

That worked. Also, according to the policy, karydia has not only verified
that `automountServiceAccountToken` is set to false but also modified the
pod to use the configured seccomp profile:

```
kubectl --as system:serviceaccount:jane:jane -n jane get pods busybox -o jsonpath='{.metadata.annotations.seccomp\.security\.alpha\.kubernetes\.io/pod}'
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
