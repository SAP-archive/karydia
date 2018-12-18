# Installing karydia

karydia can be installed as a validating and mutating admission webhook
into any Kubernetes cluster.

## Installation

### Create a TLS certificate

You can use `scripts/create-karydia-certificate` to create a certificate
signing request (openssl is used under the hood), send it to the
apiserver and let the apiserver sign the cert.

This assumes that your Kubernetes user is authorized to approve certificates.

### Create a secret with TLS certificate and key

To make the certificate available to karydia, we need to create a secret.
`./scripts/create-karydia-tls-secret` can be used for that.

### Deploy karydia

First, create a configmap that holds the default network policy that
karydia should use. Note that this is only necessary if karydia is
deployed with `--enable-default-network-policy`.

```
kubectl create configmap -n kube-system karydia-default-network-policy --from-literal policy="$(<manifests/example-default-network-policy.yml)"
```

Create the `KarydiaSecurityPolicy` CRD (not actually used currently but
expected by the controller):

```
kubectl apply -f manifests/crd-karydia-security-policy.yml
```

Create a service account with cluster-admin permissions and deploy karydia:

```
kubectl apply -f manifests/rbac.yml
kubectl apply -f manifests/deployment.yml
kubectl apply -f manifests/service.yml
```

Finally, configure karydia as both a validating and mutating admission
controller with the apiserver:

```
./scripts/configure-karydia-webhook
```

Take a quick look at the logs to see if karydia started:

```
kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f -c karydia

time="2018-11-09T10:47:50Z" level=info msg="Listening on 0.0.0.0:33333"
[...]
```

## Development and test environment

[Minikube](https://kubernetes.io/docs/setup/minikube/) can be used as
a development environment.

```
minikube start --kubernetes-version v1.13.1
eval $(minikube docker-env)
make container
```

Follow the installation instructions above.

After that, you can repeat the following commands to rebuild and test changes:

```
kubectl delete -f manifests/karydia.yml || true
eval $(minikube docker-env)
make container
kubectl apply -f manifests/karydia.yml
```

These steps allow you to test new code without having to push the container
image to a registry at each iteration.

## Installation in a [Gardener](https://github.com/gardener/gardener) setup

In a Gardener setup, you might want to install karydia in both the shoot
cluster and in the shoot's control plane (i.e. the shoot's namespace in
the seed cluster). Installation instructions for the latter differ from
the general installation and are explained below (plus some additional
configuration of demo OPA policies).

Two independent karydia installations make sense if you want to separate
concerns, for example to split policies configured by the operator
and policies configured by a cluster admin.

Note: this is for development only and work in progress.

### Setup and configuration in a shoot's control plane

Ensure you have a kubeconfig for both the seed cluster and the shoot cluster.
- `export KUBECONFIG_SHOOT=$HOME/shoot.kubeconfig`
- `export KUBECONFIG_SEED=$HOME/seed.kubeconfig`

Ensure you know:
- the name of the namespace used for the shoot cluster: `export SHOOT_NS=shoot--foo--bar`
- the CIDR used by nodes. You can find it with:

```
kubectl --kubeconfig=$KUBECONFIG_SEED -n $SHOOT_NS get configmap secvictic.infra.tf-config -o yaml | grep ip_cidr_range
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
NAMESPACE=$SHOOT_NS ./contrib/gardener/scripts/deploy-karydia-cp
RESTRICTEDENDPOINTCIDR='{"10.242.0.0/16", "10.243.0.0/16", "169.254.169.254/32"}' NODECIDR=10.250.0.0/19 PODCIDR=100.96.0.0/11 CLUSTERNAME=$SHOOT_NS NAMESPACE=$SHOOT_NS ./contrib/gardener/scripts/deploy-karydia-cp

export KUBECONFIG=$KUBECONFIG_SHOOT
./contrib/gardener/scripts/configure-karydia-webhook-cp
```
