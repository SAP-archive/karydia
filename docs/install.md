# Installing karydia

karydia can be installed in several ways:
- Development and test environment with minikube
- Gardener shoot cluster
- In the control plane of a Gardener shoot cluster

## Development and test environment

First, install [minikube](https://kubernetes.io/docs/setup/minikube/).

Then the following commands can be used to compile and install karydia. This includes:
- a deployment
- a service
- a config map for the default network policy
- a default KarydiaSecurityPolicy
- mutating and validating webhooks

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

## Gardener shoot cluster

First, ensure `kubectl` has acces to your shoot cluster.

Then, the following commands will install a pre-compiled version of karydia:

```
rm karydia*pem
./scripts/create-karydia-certificate
./scripts/create-karydia-tls-secret

kubectl apply -f manifests/deployment.yml
kubectl apply -f manifests/service.yml

kubectl create configmap -n kube-system karydia-default-network-policy --from-literal policy="$(<manifests/example-default-network-policy.yml)"

kubectl apply -f manifests/crd-karydia-security-policy.yml

./scripts/configure-karydia-webhook
```

## In the control plane of a Gardener shoot cluster

TODO
