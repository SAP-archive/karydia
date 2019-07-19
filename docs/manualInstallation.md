# Install karydia by hand
It is possible to install karydia manually without the need of Helm and Tiller. This process is quite more complex and needs patience. If you just want to install karydia have a look at the [installation with Helm](../install/README.md).

The manual process consists of the following steps:
- Prepare the manual installation
- Create a TLS certificate and secret
- Deploy the manifests
- Configure the webhook

## Prepare Manual Installation

## Create a TLS Certificate and Secret
To create a certificate signing request run:
```
scripts/create-karydia-certificate
```

As the certificate should be available to karydia, create a secret by running:
```
scripts/create-karydia-tls-secret
```

## Deploy Manifests
First, register the karydia config custom resource definition (CRD) followed by the creation of a karydia config custom resource that holds the karydia default config which should be used.
```
kubectl apply -f manifests/crd-config.yml
kubectl apply -f manifests/config.yml
```

After that, create a configmap that holds the default network policy that karydia should use. Note that this is only necessary if karydia is deployed with `--enable-default-network-policy`.
```
kubectl create configmap -n kube-system karydia-default-network-policy --from-literal policy="$(<manifests/example-default-network-policy.yml)"
```

Last, create a service account for karydia and deploy the following manifests:
```
kubectl apply -f manifests/namespace.yml
kubectl apply -f manifests/rbac.yml
kubectl apply -f manifests/deployment.yml
kubectl apply -f manifests/service.yml
```

## Configure Webhook
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
