# Installation
karydia can be installed as a validating and mutating admission webhook into any Kubernetes cluster.

The installation processes uses helm and tiller to get karydia up and running. In addition, the needed certificates and secretes are genereated.

## Helm and Tiller
To install karydia you have to provide Tiller on the cluster. Thus, run (https://github.com/istio/istio/blob/master/install/kubernetes/helm/helm-service-account.yaml):
```
kubectl apply -f helm-service-account.yaml
```
and afterwards execute:
```
helm init --service-account tiller
```

## Certificates and Secrets
In the next step, you have to create the needed certificate and configure some options by running:
```
./charts/prepare-karydia-install
```
This creates a certificate signing request, sends it to the apiserver and lets the apiserver sign the cert. To make the certificate available to karydia, a secret is created.

It will also create a configmap for the default network policy and writes the CA bundle for the later established webhook into teh values.yml.

## Deploy karydia
To deploy karydia simply run:
```
helm install ./charts --name karydia
```

# Make sure that karydia is runnning
Afterwards, you can check if karydia is running by taking a look at the logs:
```
kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f -c karydia

time="2018-11-09T10:47:50Z" level=info msg="Listening on 0.0.0.0:33333"
[...]
```
