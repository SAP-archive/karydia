# Karydia - A Kubernetes Security Walnut
Karydia is a security add-on for Kubernetes, which helps you follow good security practices by inverting insecure default settings in Kubernetes. Kubernetes default settings are not optimized for security, but rather on running out-of-the-box without complicated configuration upfront. It's easy to get a pod up and running; in the simplest case it's just one command. Unfortunately, the simple setup does not have a highly secure application in mind. Default settings are not enough!

Karydia inverts the following insecure default settings:
* Unmount service account token
* Restrict system calls by adding a seccomp profile
* Run with minimal privileges by adding a non-root user
* Restrict network communication by adding a network policy to each namespace

A description of each feature can be found [here](./features.md) and an overview of the application of these features is described in the [demo section](./demos/overview.md). This project provides some demos providing instructions for the features provided by Karydia. This includes:
- A basic tutorial about [seccomp profile](./demos/seccomp.md) and instructions to use [custom seccomp profiles](custom_seccomp.nd).

## Installing Karydia
To install Karydia using Helm run the following commands:
```
kubectl apply -f ./install/helm-service-account.yaml
helm init --service-account tiller
helm install ./install/charts --name karydia --namespace karydia
```

A detailed description of the installation process can be found in the [corresponding readme](https://github.com/karydia/karydia/blob/master/install/README.md).

## Development
For developers we provide an [architectural overview](./devel/architecture.md) and instructions to use a [hot swap technology](./devel/hotswap.md).
