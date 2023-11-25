Tekton Pipelines Feedback
=========================

Kubernetes controller - provides an SCM (and not only) integration to notify external systems about Tekton Pipelines execution.


Built using pipelines-feedback-core
-----------------------------------

pipelines-feedback-core is a universal framework for integrating any Kubernetes-based job executor with external systems.

Getting started
---------------

Pipelines Feedback Tekton is available as a Helm Chart for installation.

### Using ArgoCD

Simply import it as a Helm application.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: pipelines-feedback-tekton
  namespace: cluster-root
spec:
  destination:
    namespace: my-namespace
    server: https://kubernetes.default.svc
  project: default
  source:
    chart: tekton-chart
    helm:
      values: |
        rbac:
            resourceNames: ["my-secret-name-in-every-namespace"]
    repoURL: quay.io/pipelines-feedback
    targetRevision: 0.1
  syncPolicy: {}
```

### Manually with Helm

```bash
helm install pft oci://quay.io/pipelines-feedback/tekton-chart --version 0.0.1-latest-main --dry-run
```
