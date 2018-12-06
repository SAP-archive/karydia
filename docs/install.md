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

kubectl apply -f manifests/rbac.yml
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

First, ensure `kubectl` has access to your shoot cluster.

Then, the following commands will install a pre-compiled version of karydia:

```
rm karydia*pem
./scripts/create-karydia-certificate
./scripts/create-karydia-tls-secret

kubectl apply -f manifests/rbac.yml
kubectl apply -f manifests/deployment.yml
kubectl apply -f manifests/service.yml

kubectl create configmap -n kube-system karydia-default-network-policy --from-literal policy="$(<manifests/example-default-network-policy.yml)"

kubectl apply -f manifests/crd-karydia-security-policy.yml

./scripts/configure-karydia-webhook
```

## In the control plane of a Gardener shoot cluster

Note: the steps below are for development only. This is a work in progress.

Ensure you have a kubeconfig for both the seed cluster and the shoot cluster.
- `export KUBECONFIG_SHOOT=$HOME/shoot.kubeconfig`
- `export KUBECONFIG_SEED=$HOME/seed.kubeconfig`

Ensure you know:
- the name of the namespace used for the shoot cluster: `export SHOOT_NS=shoot--foo--bar`
- the CIDR used by nodes. You can find it with:
```
kubectl --kubeconfig=$KUBECONFIG_SEED -n $SHOOT_NS get configmap secvictic.infra.tf-config -o yaml|grep ip_cidr_range
      ip_cidr_range = "10.250.0.0/19"
```
- the CIDR used by pods. You can find it with:
```
kubectl get deploy -n $SHOOT_NS kube-controller-manager -o yaml|grep cluster-cidr
        - --cluster-cidr=100.96.0.0/11
```
- the CIDRs used in the seed cluster that you don't want the shoot cluster to meddle.

```
rm karydia*pem

export KUBECONFIG=$KUBECONFIG_SHOOT
./scripts/create-karydia-certificate

export KUBECONFIG=$KUBECONFIG_SEED
NAMESPACE=$SHOOT_NS ./scripts/create-karydia-tls-secret
NAMESPACE=$SHOOT_NS ./scripts/deploy-karydia-cp
RESTRICTEDENDPOINTCIDR='{"10.242.0.0/16", "10.243.0.0/16", "169.254.169.254/32"}' NODECIDR=10.250.0.0/19 PODCIDR=100.96.0.0/11 CLUSTERNAME=$SHOOT_NS NAMESPACE=$SHOOT_NS ./scripts/deploy-karydia-cp

export KUBECONFIG=$KUBECONFIG_SHOOT
./scripts/configure-karydia-webhook-cp
```

