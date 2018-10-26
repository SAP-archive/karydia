# karydia - Kubernetes security walnut

## Development environment

```
minikube start --kubernetes-version v1.12.2

rm karydia*pem
./scripts/create-karydia-certificate
./scripts/create-karydia-tls-secret

eval $(minikube docker-env)

make container

kubectl apply -f manifests/karydia.yml

kubectl apply -f manifests/crd-karydia-security-policy.yml

kubectl apply -f manifests/example-karydia-security-policy.yml

./scripts/configure-karydia-webhook

kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f
```

## Testing

With the example `KarydiaSecurityPolicy` applied, creating a regular pod
should fail:

```
kubectl run --rm -ti --restart=Never --image busybox busybox
Error from server (InternalError): Internal error occurred: admission webhook "karydia.gardener.cloud" denied the request: map[example-ksp:[[automount of service account 'default' not allowed]]]
```

Creating a pod with a custom service account should fail as well:

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
Error from server (InternalError): error when creating "STDIN": Internal error occurred: admission webhook "karydia.gardener.cloud" denied the request: map[example-ksp:[[automount of service account not allowed]]]
```

Let's update the KSP to allow `non-default` service accounts:

```
cat <<EOF | kubectl apply -f -
apiVersion: karydia.gardener.cloud/v1alpha1
kind: KarydiaSecurityPolicy
metadata:
  name: example-ksp
spec:
  automountServiceAccountToken: "non-default"
EOF
```

And try to start the busybox pod again.
