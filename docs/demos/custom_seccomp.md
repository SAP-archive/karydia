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

## Deploying a custom seccomp profile

## Setting up karydia with custom seccomp profile
