# Karydia development with hotswap in running container

## Why

During development it is easier and faster just to build the go binary locally
and manually test it directly in an already running Kubernetes Karydia
(development) deployment in the cloud rather than building and pushing a new
Docker image and deploy it to Kubernetes as well.

## Prerequisite
Make sure your kubectl cli talks to your desired Kubernetes cluster

## [OPTIONAL] Build

Build the dev Karydia container:
```
make container-dev
```
and push it to your docker registry.

## <a name="getting-started"></a> Getting started

Follow the steps of [installing Karydia](../install/README.md) using the parameter `--set dev.active=true`.

Develop a new feature.

### Live testing

Build Karydia and copy this new Karydia go binary directly into the running cloud dev container:
```
make build deploy-dev
```

### Live debugging

Build Karydia with debug symbols and copy this new Karydia go binary directly into the running cloud dev container:
```
make build-debug deploy-dev
```

Forward debug port to `localhost` via:
```
make debug-dev
```

Connect code debugger, e.g. from IDE, to (remote) debug port (default: `localhost:2345`)

## FAQ

### - Did the hotswap script noticed the newly copied / uploaded Karydia binary and did it restart Karydia?

Please monitor the Kubernetes (K8s) Karydia(-dev) pod logs while running `make deploy-dev`, e.g. with:
```
kubectl logs -f -n kube-system $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -c karydia | grep 'karydia-dev'
```

#### If there are outputs like the following:
```
DATE                   	TYPE	USER  	FILE       	MESSAGE           	EVENTS
YYYY-MM-DD HH:MM:SS UTC	INFO	root  	karydia-dev	killed & restarted	CLOSE_WRITE,CLOSE
```
Issue & solution: __everything seems to work as expected - nothing else to do__

#### If there are outputs like the following:
```
DATE                   	TYPE	USER  	FILE       	MESSAGE           	EVENTS
YYYY-MM-DD HH:MM:SS UTC	ERR	root  	karydia-dev	never freed file  	CLOSE_WRITE,CLOSE
YYYY-MM-DD HH:MM:SS UTC	ERR	root  	karydia-dev	procs never ended 	CLOSE_WRITE,CLOSE
```
Issue & solution: __waiting timeout issue occured - try again or try to increase timeouts manually__

1. add script parameter `-t` with a desired increase value, e.g. `5` by using `helm upgrade --set dev.active=true --set dev.timeoutIncreaseValue=5`.

2. deploy changes and try `make deploy-dev` again

Possible reason: failing / incomplete `kubectl cp` or `karydia` shutdown takes too long

#### If there are no outputs

Issue & solution: __you may use an incomplete Karydia installation or a wrong deployment, e.g. did not set `--set dev.active=true` - start from scratch and jump to [Getting started](#getting-started)__

### - Where are the files located within the Karydia container?

With the following command it is possible to connect to bash terminal within the running Karydia container:
```
kubectl exec -n kube-system -it $(kubectl get pods -n kube-system -l app=karydia -o jsonpath='{.items[0].metadata.name}') -- /bin/bash
```

File            | Location                           | Description
--------------- | ---------------------------------- | ---------------------------------
hotswap-dev.log | /go/src/github.com/karydia/karydia | hotswap logs like the ones mentioned above, e.g. infos about restarts and / or errors
karydia.log     | /go/src/github.com/karydia/karydia | potentially some additional logs from specified run command
hotswap-dev     | /usr/local/bin                     | hotswap-dev script bound to main container process
karydia         | /usr/local/bin                     | Karydia binary called from hotswap-dev script
karydia-dev     | /usr/local/bin                     | Karydia-dev binary copied / uploaded via `kubectl cp` - creation / modification of this file triggers hotswap routine; This file only exists for a short period of time (between `kubectl cp` and hotswap routine start) because it gets renamed as `karydia` and, thus, replaces the old `karydia` file.

