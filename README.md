# karydia - Kubernetes security walnut

Status: alpha, work in progress

karydia is a security add-on to Kubernetes to help with good security practices
and assist administrators in keeping their clusters safe. karydia is implemented as
[webhook admission
controller](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
and configurable through its command line interface and Kubernetes resources.

While it has been written with [Gardener](https://gardener.cloud/) in mind, it
can be used on any Kubernetes cluster.

![](docs/images/karydia-architecture.png)

## Installing karydia

See [installing karydia](docs/install.md).

## Demo

Please see [docs/demo.md](docs/demo.md) for a demo of some karydia features.

## Features and configuration options

See [features](docs/features.md).

## Testing

### Integration tests

```
minikube start --kubernetes-version v1.12.3
eval $(minikube docker-env)
make container
make e2e-test
```

### Unit tests

```
make test
```
