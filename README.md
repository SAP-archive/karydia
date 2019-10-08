# Karydia - A Kubernetes Security Walnut

![Karydia Logo](logo/Karydia@0.5x.png)

Status: Beta

Karydia is a security add-on for Kubernetes, which helps you follow good security practices by inverting insecure default settings in Kubernetes. Kubernetes default settings are not optimized for security, but rather on running out-of-the-box without complicated configuration upfront. It's easy to get a pod up and running; in the simplest case it's just one command. Unfortunately, the simple setup does not have a highly secure application in mind. Default settings are not enough!

Karydia inverts the following insecure default settings:
* Unmount service account token
* Restrict system calls by adding a seccomp profile
* Run with minimal privileges by adding a non-root user
* Restrict network communication by adding a network policy to each namespace

Karydia is implemented as [webhook admission
controller](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
and is configurable through a custom Kubernetes resource.

![](docs/images/karydia-architecture.png)

## Installing Karydia
[Install](install/README.md) Karydia with a helm chart.

## Demo
### Invert the Pod Defaults
The following screenshot shows the pod specification without the usage of Karydia:
* Service accout token is mounted
* A user is not specified (so the pod uses root by default)
* No seccomp profile is assigned
```
kubectl run -it --rm --restart=Never alpine --image=alpine sh -n demo
kubectl edit  pod/alpine -n demo
```
![](docs/images/pod-without-karydia.png)

If you create a pod after the installation of Karydia, the pod description is different, even if you use the same commands:
* No service account token is mounted
* A user is specified (the root user is not used)
* The seccomp profile runtime/default is assigned

```
kubectl run -it --rm --restart=Never alpine --image=alpine sh -n demo
kubectl edit  pod/alpine -n demo
```
![](docs/images/pod-with-karydia.png)

### Add a Network Policy
Karydia adds a default network policy to each namespace and reconciles it.
```
kubectl get networkpolicy -n demo
```
![](docs/images/networkpolicy.png)

## Features and Configuration Options
You can configure each feature to meet the needs of your applications:
* A custom seccomp profile
* A custom default network policy
* A specific network policy per namespace
* The usage of a root user if necessary

See all [features and options](docs/features.md).

## Testing

### Integration Tests

##### Install Karydia Dev
```
kubectl apply -f ./install/helm-service-account.yaml
helm init --service-account tiller
helm install ./install/charts --name karydia --namespace karydia --set dev.active=true
```

##### Build, Swap and Test

```
make build deploy-dev
make e2e-test
```

### Unit Tests

```
make test
```

### Debug Karydia

To debug (for example Visual Studio Code), change the following line in the debug configuration:

```
"args": ["--kubeconfig","<PATH>/.kube/config"]
```
