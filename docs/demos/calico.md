# Calico - Policy-driven network security

## Basics

### Kubernetes
Kubernetes defines `NetworkPolicies` as a distinct resource. A `NetworkPolicy` is a specification of how groups of pods are allowed to communicate with each other and other network endpoints. However, Kubernetes does not manage nor enforce these `NetworkPolicies`. This is implemented by the network plugin/solution. There are many different network solution, including Calico (which is already installed by Gardener into every cluster).

Sources: 
- https://kubernetes.io/docs/concepts/services-networking/network-policies/
- https://kubernetes.io/docs/concepts/cluster-administration/networking/

### Calico
TO-DO

### NetworkPolicy and GlobalNetworkPolicy
Calico distinguished between `NetworkPolicies` and `GlobalNetworkPolicies`.

Both are resources that represent an ordered set of rules which are applied to a collection of endpoints that match a label selector.

A `NetworkPolicy`is only defined for a single namespace. It only applies to workload endpoint resources in that namespace. In contrast, a `GlobalNetworkPolicy`is not a namespaced resource. A `GlobalNetworkPolicy` applies to workload endpoint resources in all namespaces, and to host endpoint resources. 

Sources:
- https://docs.projectcalico.org/v3.9/reference/resources/networkpolicy
- https://docs.projectcalico.org/v3.9/reference/resources/globalnetworkpolicy

## Installation
TO-DO

## Demo

IN PROGRESS!
