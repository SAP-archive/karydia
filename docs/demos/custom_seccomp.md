# How to create and deploy a custom seccomp profile
## Creating a custom seccomp profile
Seccomp profiles are json files that define which system calls are allowed or prohibited. It consists of the `defaultAction` that should be applied for all system calls that are not specified concretely, the target architectures(`architectures`) and specific rules for one or multiple system calls (`syscalls`).

An example of a custom seccomp profile that allows all system calls (stated by `"defaultAction": "SCMP_ACT_ALLOW"`) and only prohibits the `chmod` system call.
```
{
	"defaultAction": "SCMP_ACT_ALLOW",
	"architectures": [
		"SCMP_ARCH_X86_64",
		"SCMP_ARCH_X86",
		"SCMP_ARCH_X32"
	],
	"syscalls": [
		{
			"names": [
				"chmod"
			],
			"action": "SCMP_ACT_ERRNO",
			"args": [],
			"comment": "",
			"includes": {},
			"excludes": {}
		}
	]
}
```
For a more technical insight, have a look at the [linux programmer's handbook](http://man7.org/linux/man-pages/man2/seccomp.2.html) and more examples can be found at the [docker repository](https://github.com/docker/labs/tree/master/security/seccomp/seccomp-profiles).

## Save the custom seccomp profile on the cluster
To make the custom seccomp profile available, you have to place it at `/var/lib/kubelet/seccomp` on the cluster nodes. This can be achieved using `daemonSet` which mounts `/var/lib/kubelet/seccomp` and copies the profile to this folder. A deamon set that does all this steps could look like this:
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: seccomp-profile
  namespace: kube-system
data:
  custom-seccomp.json: |
    {
      "defaultAction": "SCMP_ACT_ALLOW",
      "architectures": [
        "SCMP_ARCH_X86_64",
	"SCMP_ARCH_X86",
	"SCMP_ARCH_X32"
      ],
      "syscalls": [
	{
	  "names": [
	  "chmod"
	],
	  "action": "SCMP_ACT_ERRNO",
	  "args": [],
	  "comment": "",
	  "includes": {},
	  "excludes": {}
	}
      ]
  }

---

apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: seccomp-deamon
  namespace: kube-system
spec:
  template:
    spec:
      initContainers:
      - name: installer
        image: alpine:3.10.0
        command: ["/bin/sh", "-c", "cp -r -L /seccomp/*.json /host/seccomp/"]
        volumeMounts:
        - name: profiles
          mountPath: /seccomp
        - name: hostseccomp
          mountPath: /host/seccomp
          readOnly: false
      containers:
      - name: pause
        image: k8s.gcr.io/pause:3.1
      terminationGracePeriodSeconds: 5
      volumes:
      - name: hostseccomp
        hostPath:
          path: /var/lib/kubelet/seccomp
      - name: profiles
        configMap:
          name: seccomp-profile
```

## Activate the custom seccomp profile
To bind a specific seccomp profile to a pod/container, you can use the following alpha annotations:
- Specify a Seccomp profile for all containers of the Pod: ```seccomp.security.alpha.kubernetes.io/pod```
- Specify a Seccomp profile for an individual container: ```container.seccomp.security.alpha.kubernetes.io/${container_name}```

For the value of the annotation you can use on of the following contents:

| **Value**                    | **Description**                                                    |
|--------------------------|----------------------------------------------------------------|
| runtime/default          | the default profile for the container runtime                  |
| unconfined               | unconfined profile, disable Seccomp sandboxing                 |
| localhost/profile-name | the profile installed to the node’s local seccomp profile root |

In our case, a pod with the created custom profile would look like this:
```
apiVersion: v1
kind: Pod
metadata:
  name: testPod
  annotations:
    seccomp.security.alpha.kubernetes.io/pod: localhost/custom-seccomp
    container.seccomp.security.alpha.kubernetes.io/testContainer: localhost/custom-seccomp
spec:
  containers:
  - name: testContainer
    image: k8s.gcr.io/pause:3.1                         
```

For more inforamtion have a look at the [kubernetes documentation](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#seccomp).

## Using karydia to handle seccomp profiles
karydia makes it easy to enforce (custom) seccomp profiles on all your pods. You can configure the used profile in the `values.yaml` file by setting the `config.seccompProfile` value. You can use the same values as described in the table above. For your custom profile the value must be: `seccompProfile: "localhost/custom-seccomp"`.

karydia will take care of the rest and enforces the defined profile in all pods and containers.
