# karydia development cycle with Telepresence

## Why Telepresence

During the development cycle, it is faster to run the karydia process on your
computer directly rather than building and deploying a Docker image on
Kubernetes. However, to test karydia with cloud provider resources (nodes,
volumes, load balancers), we still need to have a Kubernetes cluster installed
in the cloud.

This is where Telepresence helps us to have the best of both worlds.
- opa and kube-mgmt will start in the cloud
- karydia will start on your computer

## Getting started

Prepare certificates:
```
export KUBECONFIG=$HOME/cluster.kubeconfig
rm karydia*pem
./scripts/create-karydia-certificate
./scripts/create-karydia-tls-secret
```

Build karydia:
```
make
```

Deploy OPA and kube-mgmt but without Karydia:
```
kubectl apply -f manifests-dev/karydia-dev-telepresence.yml
./scripts/configure-karydia-webhook
```

Install some examples of policies
```
./scripts/push-opa-policies
```

Start karydia on your laptop:
```
telepresence --namespace kube-system --deployment karydia --expose 33333:8000 --also-proxy opa --run-shell
bin/karydia runserver --tls-cert karydia.pem --tls-key karydia-key.pem --kubeconfig $KUBECONFIG --opa-api-endpoint http://$OPA_SERVICE_HOST/v1
```

## Get OPA data

```
export OPA_IP=$(kubectl get svc -n kube-system opa -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
curl -XPUT -d '{"content": "true"}' http://$OPA_IP/v1/data/test/foo/bar
curl http://$OPA_IP/v1/data/test?pretty=true

curl -d '{"query": "a=data.kubernetes.namespaces with data.kubernetes.namespaces.extratest as {\"foo\": 42}"}' http://$OPA_IP/v1/query?pretty=true
```

