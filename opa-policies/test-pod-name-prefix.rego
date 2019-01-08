package admission
import data.k8s.matches
import data.kubernetes.pods
forbiddenPrefix = "nonono"
deny[{
    "id": "test-pod-name-prefix",
    "resource": {"kind": "pods", "namespace": namespace, "name": name},
    "resolution": {"message" : sprintf("cannot use pod name %q", [requestedName])},
}] {
    matches[["pods", namespace, name, matched_object]]
    namePrefix = sprintf("%s-", [forbiddenPrefix])
    requestedName := matched_object.metadata.name
    startswith(matched_object.metadata.name, namePrefix)
}
