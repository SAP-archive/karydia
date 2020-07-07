# Troubleshooting Guide
Over the past, we have seen some errors occur multiple times. Thus, this document gives an overview of common erros and demonstrate how to resolve them. If you encounter any problems that are not mentioned in this guide, please open a [GitHub Issue](https://github.com/karydia/karydia/issues/new?assignees=&labels=bug&template=bug_report.md&title=).

## Hibernation/Restarting the cluster fails
**Problem:** I cannot hibernate or restart my cluster while Karydia is installed.

**Solution:** Make sure that you are running the [newest version of Karydia](https://github.com/karydia/karydia/blob/master/install/charts/Chart.yaml#L20). If this does not fix the problem, do the following: (1) uninstall Karydia, (2) restart/hibernate the cluster, and (3) reinstall Karydia.

## Pod/Container does not start after installing Karydia
**Problem:** A pod/container does not start (or work properly) after I installed Karydia.

**Solution:** Make sure that the pod/container is runnning with the suffcient permission (e.g. runs a user with sufficient permission, is allowed to call another pod/container). Karydia will not overwrite explicitly set values.

For example, while using [nginx](https://github.com/nginx/nginx) you have to explicitly set a privileged user (e.g. user/group `0`):
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec
  [...]
  spec:
    securityContext:
      runAsUser: 0
      runAsGroup: 0 
    containers:
    - name: nginx
      image: nginx:latest
      ports:
      - containerPort: 80
```
