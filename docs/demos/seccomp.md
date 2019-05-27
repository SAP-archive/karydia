# Seccomp Demo

## Create a Pod without seccomp profile
You can create a simple pod and execute an interacitve shell in it by running:
```
kubectl run --rm -i --tty busybox --image=busybox -- sh
```

Next, show that no seccomp profile is deployed ("0 = SECCOMP_MODE_DISABLED"):
```
grep Seccomp /proc/1/status

=> Seccomp:	0
```

## Try to run "unshare" syscall on unsecure pod
First, show the current user:
```
whoami

=> root
```

Afterwards, try to use an unshared namspace:
```
unshare --user whoami

=> nobody
```

## Install karydia
See [install instructions](/install/README.md) for more information.

## Create a pod with seccomp profile enforced by karaydia
You can create a simple pod and execute an interacitve shell in it by running:
```
kubectl run --rm -i --tty busybox --image=busybox -- sh
```

Next, show that default seccomp profile is deployed ("2 = SECCOMP_MODE_FILTER"):
```
grep Seccomp /proc/1/status

=> Seccomp:	2
```

## Try to run "unshare" syscall on secure pod
First, show the current user:
```
whoami

=> root
```

Afterwards, try to use an unshared namspace:
```
unshare --user whoami

=> Operation not permitted
```

# Overview of "docker/default" seccomp
An overview of the not permitted sycalls can be found on the docker [website](https://docs.docker.com/engine/security/seccomp/).
