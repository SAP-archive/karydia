# karydia development with hotswap in running container

## Why

During development it is easier and faster just to build the go binary locally
and manually test it directly in an already running Kubernetes karydia
(development) deployment in the cloud rather than building and pushing a new
Docker image and deploy it to Kubernetes as well.

## Prerequisite
Make sure your kubectl cli talks to your desired Kubernetes cluster

## [OPTIONAL] Build

Build the dev karydia container:
```
make container-dev
```
and push it to your docker registry.
Adjust `manifests-dev/deployment-dev.yml` to your docker registry image manually or just use the
next step.

## [OPTIONAL] Re-Generate the `manifests-dev/deployment-dev.yml`

You can use `scripts/generate-deployment-dev` to automatically re-generate the
`manifests-dev/deployment-dev.yml` from `manifests/deployment.yml`.

Two parameters are expected:
- PROD_DOCKER_IMAGE which is the mentioned image under the karydia container spec at
`manifests/deployment.yml` (e.g. `karydia/karydia`)
- DEV_DOCKER_IMAGE which is the dev container image (e.g. `karydia/karydia-dev` OR your docker
registry image from the previous step)
```
scripts/generate-deployment-dev karydia/karydia karydia/karydia-dev
```

## Getting started

Follow the steps of [installing karydia](../install.md)

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

