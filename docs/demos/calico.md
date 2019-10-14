# Calico - Policy-driven network security

## Basics

### Kubernetes
Kubernetes defines `NetworkPolicies` as a distinct resource. A `NetworkPolicy` is a specification of how groups of pods are allowed to communicate with each other and other network endpoints. However, Kubernetes does not manage nor enforce these `NetworkPolicies`. This is implemented by the network plugin/solution. There are many different network solution, including Calico (which is already installed by Gardener into every cluster).

Sources: 
- https://kubernetes.io/docs/concepts/services-networking/network-policies/
- https://kubernetes.io/docs/concepts/cluster-administration/networking/

### Calico
Project Calico is a network solution for Kubernetes. Calico configures a layer 3 network that uses the BGP routing protocol to route packets between hosts. This means that packets do not need to be wrapped in an extra layer of encapsulation when moving between hosts. The BGP routing mechanism can direct packets natively without an extra step of wrapping traffic in an additional layer of traffic.

![](../images/CalicoArchitecture.jpg)

Besides the performance that this offers, one side effect of this is that it allows for more conventional troubleshooting when network problems arise.

Sources: 
- https://www.projectcalico.org/
- https://rancher.com/blog/2019/2019-03-21-comparing-kubernetes-cni-providers-flannel-calico-canal-and-weave/
- https://www.slideshare.net/packethost/intro-to-project-calico-a-pure-layer-3-approach-to-scaleout-networking

### NetworkPolicy and GlobalNetworkPolicy
Calico distinguished between `NetworkPolicies` and `GlobalNetworkPolicies`.

Both are resources that represent an ordered set of rules which are applied to a collection of endpoints that match a label selector.

A `NetworkPolicy`is only defined for a single namespace. It only applies to workload endpoint resources in that namespace. In contrast, a `GlobalNetworkPolicy`is not a namespaced resource. A `GlobalNetworkPolicy` applies to workload endpoint resources in all namespaces, and to host endpoint resources. 

Sources:
- https://docs.projectcalico.org/v3.9/reference/resources/networkpolicy
- https://docs.projectcalico.org/v3.9/reference/resources/globalnetworkpolicy

### HostEndpoints
A `hostEndpoint` are the network interfaces of the host itself. It is an interface attached to a host that is running Calico You can use host endpoint policy to secure a NAT gateway or router. Calico supports selector-based policy when running on a gateway or router, allowing for rich, dynamic security policy based on the labels attached to your host endpoints.

Source: https://docs.projectcalico.org/v3.8/security/host-endpoints/

## Installation
As Gardener already installs Calico into every cluster, there is not much to do.

On the one hand, you can create a `NetworkPolicy` policy using `kubectl` and a `GlobalNetworkPolicy` (as well as other Calico-specific resources) using `calicoctl`.

Sources:
- https://docs.projectcalico.org/v3.9/getting-started/calicoctl/install
- https://docs.projectcalico.org/v2.0/reference/calicoctl/resources/hostendpoint

### Demo
For testing Calico, we want to achieve the following setup:
- Two namespaces with each running a single pod
- ~~Predefined `HostEndpoints`~~
- A `GlobalNetworkPolicy` that applies to both namespaces and denies all traffic
- A `NetworkPolicy` that applies only to one namespace and allows all traffic by overwritting the `GlobalNetworkPolicy`

Let us start!
First, create the test environment.
Create two namespaces:
```
kubectl create namespace test1
kubectl create namespace test2
```

---

Next, set up the `NetworkPolicies` ~~with predefined `HostEndpoints`~~.
Create a `ServiceAccount` with sufficient permissions:
```
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-test
  namespace: default
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: calico-role
rules:
- apiGroups: ["crd.projectcalico.org"]
  resources: ["*"]
  verbs: ["create", "delete", "get", "patch", "update", "list"]
- apiGroups: ["networking.k8s.io"]
  resources: ["networkpolicies"]
  verbs: ["create", "delete", "get", "patch", "update", "list"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: calico-binding
subjects:
- kind: ServiceAccount
  name: calico-test
  namespace: default
roleRef:
  kind: ClusterRole
  name: calico-role
  apiGroup: rbac.authorization.k8s.io
EOF
```


Get a Shell instance running a pod:
```
kubectl run calico-shell --rm -i --tty --image ubuntu --serviceaccount calico-test -- bash
```

Install `calicoctl` into the pod's environment:
```
apt update
apt install curl
curl -O -L  https://github.com/projectcalico/calicoctl/releases/download/v3.5.8/calicoctl
chmod +x calicoctl
export DATASTORE_TYPE=kubernetes
```

~~Create a `HostEndpoint`:~~
```
n/a
```

Create a `NetworkPolicy`that allows trafic for all pods in namesapce `test1`:
```
./calicoctl create -f - <<EOF
apiVersion: projectcalico.org/v3
kind: NetworkPolicy
metadata:
  name: allow-all-low
  namespace: test1
spec:
  order: 1
  ingress:
  - action: Allow
  egress:
  - action: Allow
EOF
```

Create a `GlobalNetworkPolicy`that denies all trafic:
```
./calicoctl create -f - <<EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: deny-all
spec:
  selector: projectcalico.org/namespace in {'test1', 'test2'}
  order: 5
  ingress:
  - action: Deny
  egress:
  - action: Deny
EOF
```

---

Last, check the functionality of the implemented policies.

Run a pod in namespace `test1`:
```
kubectl run test-shell1 --rm -i --tty --image ubuntu --namespace test1 -- bash
```
and try to connect to a server (which should be allowed):
```
apt update
apt install fping
fping google.com

=> google.com is alive
```

Run a pod in namespace `test2`:
```
kubectl run test-shell2 --rm -i --tty --image ubuntu --namespace test2 -- bash
```
and try to connect to a server (which should be denied):
```
apt update

=> Err:1 http://archive.ubuntu.com/ubuntu bionic InRelease                  
  Temporary failure resolving 'archive.ubuntu.com'
Err:2 http://security.ubuntu.com/ubuntu bionic-security InRelease        
  Temporary failure resolving 'security.ubuntu.com'
Err:3 http://archive.ubuntu.com/ubuntu bionic-updates InRelease          
  Temporary failure resolving 'archive.ubuntu.com'
Err:4 http://archive.ubuntu.com/ubuntu bionic-backports InRelease
  Temporary failure resolving 'archive.ubuntu.com'
```

## Implementation Idea
Implementing Calico-based host-security policies into Karydia needs the following steps:
1. Create a service account with sufficient permission for creating the defined policies
2. Run a pod with `calicoctl` that applies the policies and other calico-specific resources
3. Kill the pod after it is finished
4. Remove all calico-related resources created by Karydia if Karydia is deleted
