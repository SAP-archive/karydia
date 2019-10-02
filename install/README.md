# Installation
karydia can be installed as a webhook into any Kubernetes cluster.

The installation processes uses [Helm](https://github.com/helm/helm) to get karydia up and running and is split up into two parts.

You can also do an [installation by hand](../docs/manualInstallation.md) without Tiller.

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
kubectl logs -n karydia $(kubectl get pods -n karydia -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f -c karydia
```

karydia is up and running, if the following information is logged:
```
{"component":"server","level":"info","msg":"[INFO] Listening on 0.0.0.0:33333","time":"2019-08-22T10:47:50Z"}
[...]
```

## Development
If you want to install the development image, simply run:
```
helm install ./install/charts --name karydia --set dev.active=true
```
Otherwise, you can set `dev.active` to `true` in the `values.yaml` file.
