# karydia Demo

This is a living document to demo karydia features (and thus should be updated
frequently).

If you want to follow the demo, set up a new test cluster and deploy
karydia.

![](demos/seccomp/seccomp.gif)

Other demos:
- [automount-service-automount-token](demos/automount-service-account-token/automount-service-account-token.md)

## karydia admission plugin

Let's start with running a pod in the default namespace:

```
$ kubectl run --rm -ti --restart=Never --image busybox busybox -- echo hello world
hello world
pod "busybox" deleted
```

As expected, that worked.

To "harden" the cluster, we would like to

* disallow the mount of default service account tokens into pods. Users should
  instead use distinct service accounts with a clear purpose and RBAC
  permissions / boundaries.
* enforce a seccompProfile annotation that pods should use.

For that we add two annotations to the 'default' namespace:

```
kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=non-default
kubectl annotate ns default karydia.gardener.cloud/seccompProfile=runtime/default
```

Now we try to run the hello world pod again:

```
$ kubectl run --rm -ti --restart=Never --image busybox busybox -- echo hello world
Error from server (InternalError): Internal error occurred: admission webhook "karydia.gardener.cloud" denied the request: [automount of service account 'default' not allowed]
```

karydia has recognized the annotations for the namespace where the pod
is run (`default`) and rejected the pod.

Let's try again w/ a new service account:

```
cat <<EOF | kubectl apply -f -
---

apiVersion: v1
kind: ServiceAccount
metadata:
  name: testsa

---

apiVersion: v1
kind: Pod
metadata:
  name: busybox
spec:
  serviceAccountName: "testsa"
  containers:
  - image: schu/busybox-nobody
    imagePullPolicy: IfNotPresent
    name: busybox
    command: ["sleep", "300"]
EOF
```

That worked. karydia noticed that the pod doesn't use the 'default' service
account and allowed the request.

Also, according to the namespace annotation, karydia has modified the
pod and added an annotation for seccomp:

```
kubectl get pods busybox -o jsonpath='{.metadata.annotations.seccomp\.security\.alpha\.kubernetes\.io/pod}'
```
## default network policies

Let's start with creating a new namespace and attempt to send network traffic:

```
kubectl create namespace test51
kubectl run --rm -n test51 -ti --restart=Never --image appropriate/curl test1 -- curl http://www.google.com
curl: (6) Could not resolve host: www.google.com
```

The traffic was stopped by the network policy automatically installed by karydia in the new network namespace:
```
$ kubectl -n test51 get networkpolicy -o yaml
NAME                             POD-SELECTOR   AGE
karydia-default-network-policy   <none>         2m22s
```

This helps to set up sensible default. It can however be removed:
```
$ kubectl delete -n test51 networkpolicy karydia-default-network-policy
networkpolicy.extensions "karydia-default-network-policy" deleted
$ kubectl run --rm -n test51 -ti --restart=Never --image appropriate/curl test1 -- curl http://www.google.com
(html content)
```

## OPA

For demo purposes, we have an OPA policy that blocks pods named with the prefix "nonono":
```
$ kubectl run --rm -ti --restart=Never --image busybox nonono-busybox -- echo hello world
Error from server (InternalError): Internal error occurred: admission webhook "karydia.gardener.cloud" denied the request: ["cannot use pod name \"nonono-busybox\""]
```

However, other names are allowed:
```
$ kubectl run --rm -ti --restart=Never --image busybox allowed-busybox -- echo hello world
hello world
pod "allowed-busybox" deleted
```
