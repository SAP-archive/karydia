# karydia development with hotswap in running container

## Why

During development it is easier and faster just to build the go binary locally
and manually test it directly in a already running Kubernetes karydia
(development) deployment in the cloud rather than building and pushing a new
Docker image and deploy it to Kubernetes as well.

## Prerequisite
Make sure your kubectl cli talks to your desired Kubernetes cluster

## Getting started

Follow the steps of [installing karydia](../install.md)

Build the dev karydia container:
```
make container-dev
```
and push it to your private / enterprise docker registry.
Adjust `manifests-dev/deployment-dev.yml` to your docker registry image.

Switch from karydia prod to dev Kubernetes deployment:
```
kubectl delete -f manifests/deployment.yml
kubectl apply -f manifests-dev/deployment-dev.yml
```

Develop a new feature

Build karydia and copy this new karydia go binary directly into the running cloud dev container:
```
make build deploy-dev
```

