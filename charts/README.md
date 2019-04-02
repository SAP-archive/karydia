# Installation
karydia can be installed as a validating and mutating admission webhook into any Kubernetes cluster.

The installation processes uses helm and tiller to get karydia up and running. In addition, the needed certificates and secretes are genereated.

To start the installation, simply run the installation script:
```
./charts/install_karydia
```

Afterwards, you can check if karydia is running by taking a look at the logs:
```
kubectl logs -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -f -c karydia

time="2018-11-09T10:47:50Z" level=info msg="Listening on 0.0.0.0:33333"
[...]
```
