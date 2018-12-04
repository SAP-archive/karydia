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

kubectl apply -f manifests/deployment.yml
kubectl apply -f manifests/service.yml

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

## Features and configuration options

See [features](docs/features.md).
