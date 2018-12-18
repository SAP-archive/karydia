# karydia - Kubernetes security walnut

Status: alpha, work in progress

karydia is a security add-on to Kubernets to help with good security practices
and assist administrators in keeping their clusters safe. Is is implemented as
[webhook admission
controllers](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
and configurable through its command line interface and Kubernetes resources.

While it has been written with [Gardener](https://gardener.cloud/) in mind, it
can be used on any Kubernetes cluster.

![](karydia-architecture.png)

## Installing karydia

See [installing karydia](docs/install.md).

## Demo

Please see [docs/demo.md](docs/demo.md) for a demo of some karydia features.

## Features and configuration options

See [features](docs/features.md).
