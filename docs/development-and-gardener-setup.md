# Development and Gardener
=======
# Installing karydia

## Installation
Installation instruction can be found [here](https://github.com/karydia/karydia/blob/master/install/README.md).

## Development and test environment

[Minikube](https://kubernetes.io/docs/setup/minikube/) can be used as
a development environment.

```
minikube start --kubernetes-version v1.13.1
eval $(minikube docker-env)
make container
```

Follow the [installation instructions](https://github.com/karydia/karydia/blob/master/install/README.md).

After that, you can repeat the following commands to rebuild and test changes:

```
kubectl delete -n kube-system deployment karydia || true
eval $(minikube docker-env)
make container
cat install/charts/templates/deployment.yml | sed -e 's|image: karydia/karydia.*|image: karydia/karydia|' | kubectl apply -f -
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
NAME=bar
kubectl --kubeconfig=$KUBECONFIG_SEED -n $SHOOT_NS get configmap $NAME.infra.tf-config -o yaml | grep ip_cidr_range
      ip_cidr_range = "10.250.0.0/19"
```

- the CIDR used by pods. You can find it with:

```
kubectl --kubeconfig=$KUBECONFIG_SEED get deploy -n $SHOOT_NS kube-controller-manager -o yaml|grep cluster-cidr
        - --cluster-cidr=100.96.0.0/11
```

- the CIDRs used in the seed cluster that you don't want the shoot cluster to meddle.

Delete files from previous installation:
```
rm *.pem *-cp.kubeconfig
```

Save the certificate authority of the shoot cluster in a file. You could get it from the kubeconfig:
```
export KUBECONFIG=$KUBECONFIG_SHOOT
kubectl config view --raw=true -o jsonpath='{.clusters[?(@.name == "'$SHOOT_NS'")].cluster.certificate-authority-data}' | base64 -d > ca.pem
```

Generate the karydia certificate and the kubeconfig used by each container of the karydia pod:
```
export KUBECONFIG=$KUBECONFIG_SHOOT
./scripts/create-karydia-certificate
SUBJECT_NAME=karydia  NAMESPACE=$SHOOT_NS CA=ca.pem ./contrib/gardener/scripts/create-karydia-kubeconfig-cp
SUBJECT_NAME=kubemgmt NAMESPACE=$SHOOT_NS CA=ca.pem ./contrib/gardener/scripts/create-karydia-kubeconfig-cp
```

Install resources on the shoot:
```
export KUBECONFIG=$KUBECONFIG_SHOOT
kubectl apply -f install/charts/templates/namespace.yml
kubectl apply -f contrib/gardener/manifests/rbac-cp.yml
```

Install components on the seed:
```
export KUBECONFIG=$KUBECONFIG_SEED
NAMESPACE=$SHOOT_NS ./scripts/create-karydia-tls-secret
RESTRICTEDENDPOINTCIDR='{"10.242.0.0/16", "10.243.0.0/16", "169.254.169.254/32"}' NODECIDR=10.250.0.0/19 PODCIDR=100.96.0.0/11 CLUSTERNAME=$SHOOT_NS NAMESPACE=$SHOOT_NS ./contrib/gardener/scripts/deploy-karydia-cp
```

Install the admission controller configuration:
```
export KUBECONFIG=$KUBECONFIG_SHOOT
./contrib/gardener/scripts/configure-karydia-webhook-cp
```
