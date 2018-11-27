package admission

import data.k8s.matches

deny[{
    "id": "hello-world",
    "resource": {
        "kind": "pods",
        "namespace": namespace,
        "name": name
    },
    "resolution": {"message": msg},
}] {
    matches[[kind, namespace, name, matched_obj]]
    msg := sprintf("Hello world from %s", [matched_obj.metadata.name])
}
