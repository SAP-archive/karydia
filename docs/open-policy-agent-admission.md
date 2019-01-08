To test with Open Policy Agent admission, create a namespace `opa` and
add policies there.

```
kubectl create ns opa
kubectl -n opa create configmap helper-kubernetes-matches --from-file opa-policies/matches.rego
kubectl -n opa create configmap test-pod-name-prefix --from-file opa-policies/test-pod-name-prefix.rego
```
