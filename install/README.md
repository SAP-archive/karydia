# Installation
karydia can be installed as a webhook into any Kubernetes cluster.

The installation processes uses [Helm](https://github.com/helm/helm) to get karydia up and running and is split up into two parts.

## Clone Respository

Clone the repository and install the dependencies:

```
go install github.com/karydia/karydia
cd ../go/src/github.com/karydia/karydia
go install ./...
```

Afterwards, change the authentication method of Git to SSH:

In `.git/config` change the `url` setting in the `[remote "origin"]` section.

## Prepare Helm and Tiller
First, create a Helm service account and initiate Tiller on the cluster. Thus, run:
```
kubectl apply -f ./install/helm-service-account.yaml
```
and afterwards execute:
```
helm init --service-account tiller
```

## Delpoy karydia
Next, to deploy karydia simply run:
```
helm install ./install/charts --name karydia
```

## Make sure that karydia is runnning
Afterwards, you can check if karydia is running by taking a look at the logs:
```
kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f -c karydia
```

karydia is up and running, if the following information is logged:
```
time="2018-11-09T10:47:50Z" level=info msg="Listening on 0.0.0.0:33333"
[...]
```

## Development
If you want to install the development image, simply run:
```
helm install ./install/charts --name karydia --set dev.active=true
```
Otherwise, you can set `dev.active` to `true` in the `values.yaml` file.
