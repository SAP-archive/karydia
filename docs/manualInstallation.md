# Install karydia by hand
It is possible to install karydia manually without the need of Tiller. This process is quite more complex and needs patience. If you just want to install karydia have a look at the automated [installation with Helm and Tiller](../install/README.md).

The manual process consists of the following steps:
- Prepare the manual installation
- Deploy the manifests

## Prepare Manual Installation
First, generate the manifests from the helm templates:
```
mkdir manifests
helm template ./install/charts/ --output-dir manifests
```

The files for the manual installation will be stored in folder `/manifests/karydia/templates/`.

## Deploy Manifests
First, register the karydia config custom resource definition (CRD) followed by the creation of a karydia config custom resource that holds the karydia default config which should be used.

```
kubectl apply -f manifests/karydia/templates/crd-config.yaml
kubectl apply -f manifests/karydia/templates/config.yaml
kubectl apply -f manifests/karydia/templates/crd-karydia-network-policy.yaml
kubectl apply -f manifests/karydia/templates/karydia-network-policy.yaml
```

Create the namespace for karydia and configure the annotations:
```
kubectl apply -f manifests/karydia/templates/karydia-namespace.yaml
kubectl apply -f manifests/karydia/templates/workaround-annotate-kube-system-namespace.yaml
```

Create a configmap that holds the scripts for TLS/secrete creation and creates the default network policy:
```
kubectl apply -f manifests/karydia/templates/configmap.yaml
````

Last, create a service account for karydia and deploy the following manifests:
```
kubectl apply -f manifests/karydia/templates/rbac.yaml
kubectl apply -f manifests/karydia/templates/deployment.yaml
kubectl apply -f manifests/karydia/templates/service.yaml
```

Finally, configure karydia as both a validating and mutating admission controller with the API server:
```
kubectl apply -f manifests/karydia/templates/post-install-webhook.yaml
```

## Make sure that karydia is running
Take a quick look at the logs to see if karydia started:
```
kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f -c karydia

{"component":"server","level":"info","msg":"[INFO] Listening on 0.0.0.0:33333","time":"2019-08-22T10:47:50Z"}
[...]
```
