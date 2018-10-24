# karydia - Kubernetes security walnut

## Development environment

```
minikube start --kubernetes-version v1.12.1

rm karydia*pem
./scripts/create-karydia-certificate
./scripts/create-karydia-tls-secret

eval $(minikube docker-env)

make container

kubectl apply -f manifests/karydia.yml

./scripts/configure-karydia-webhook

kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f
```
