# Install karydia by hand
It is possible to install karydia manually without the need of Helm and Tiller. This process is quite more complex and needs patience. If you just want to install karydia have a look at the [installation with Helm](../install/README.md).

The manual process consists of the following steps:
- Prepare the manual installation
- Deploy the manifests
- Configure the webhook

## Prepare Manual Installation
First, generate the manifests from the helm templates:
```
helm template ./install/charts/ --output-dir manifests
```

The files for the manual installation will be stored in folder `/manifests/karydia/templates/`.

**TODO: CRD are annotaed with crd-install hook... (deleting it manually is cumbersome and error-prone)**

As the Helm installation uses post-install hooks, you have to delete the following lines from the `manifests/karydia/templates/crd-config.yaml`:
```
  annotations:
    "helm.sh/hook": crd-install
    "helm.sh/hook-delete-policy": "before-hook-creation"
```

## Deploy Manifests
First, register the karydia config custom resource definition (CRD) followed by the creation of a karydia config custom resource that holds the karydia default config which should be used.

```
kubectl apply -f manifests/karydia/templates/crd-config.yaml
kubectl apply -f manifests/karydia/templates/config.yaml
```

Create a configmap that holds the scripts for TLS/secrete creation and creates the default network policy (and for the webhook configuration):
```
kubectl apply -f manifests/karydia/templates/configmap.yaml
````

Last, create a service account for karydia and deploy the following manifests:
```
kubectl apply -f manifests/karydia/templates/namespace.yaml
kubectl apply -f manifests/karydia/templates/rbac.yaml
kubectl apply -f manifests/karydia/templates/deployment.yaml
kubectl apply -f manifests/karydia/templates/service.yaml
```

## Configure Webhook
**TODO: Is actually a post-install hook... (having a second file in the project should work for now)**

Finally, configure karydia as both a validating and mutating admission controller with the API server:
```
scripts/configure-karydia-webhook
```

## Make sure that karydia is running
Take a quick look at the logs to see if karydia started:
```
kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f -c karydia

time="2018-11-09T10:47:50Z" level=info msg="Listening on 0.0.0.0:33333"
[...]
```
