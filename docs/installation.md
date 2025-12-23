---
layout: default
title: Installation Guide
nav_order: 2
description: "Detailed installation instructions for kube-changejob"
---

# Installation Guide

This guide provides detailed instructions for installing kube-changejob in your Kubernetes cluster.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation Methods](#installation-methods)
- [Configuration](#configuration)
- [Verification](#verification)
- [Upgrading](#upgrading)
- [Uninstallation](#uninstallation)

## Prerequisites

### Required

- **Kubernetes cluster**: Version 1.29 or later
- **kubectl**: Configured to access your cluster
- **cert-manager**: For managing webhook certificates

### Optional

- **Prometheus**: For metrics collection (optional)
- **Kustomize**: For customized deployments (v5.0+)
- **Helm**: If using Helm chart (coming soon)

### Checking Prerequisites

```bash
# Check Kubernetes version
kubectl version --short

# Check cluster access
kubectl cluster-info

# Check if cert-manager is installed
kubectl get pods -n cert-manager
```

## Installation Methods

### Method 1: Quick Install (Recommended)

The quickest way to get started:

```bash
# Install cert-manager (if not already installed)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=ready pod -l app=cert-manager -n cert-manager --timeout=120s

# Install kube-changejob
kubectl apply -k github.com/nusnewob/kube-changejob/config/default

# Wait for deployment to be ready
kubectl wait --for=condition=available deployment/kube-changejob-controller-manager \
  -n kube-changejob-system --timeout=120s
```

### Method 2: Clone and Install

For more control or customization:

```bash
# Clone the repository
git clone https://github.com/nusnewob/kube-changejob.git
cd kube-changejob

# Install cert-manager (if not already installed)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml

# Install kube-changejob
make deploy

# Or use kubectl directly
kubectl apply -k config/default
```

### Method 3: Custom Kustomize Overlay

For production deployments with customization:

1. Create a kustomization overlay:

```bash
mkdir -p my-kube-changejob
cd my-kube-changejob
```

2. Create `kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: kube-changejob-system

resources:
  - https://github.com/nusnewob/kube-changejob/config/default

# Customize the deployment
patchesStrategicMerge:
  - patches/deployment.yaml

# Add custom namespace labels
namespaceLabels:
  environment: production
  team: platform

# Add resource limits
patches:
  - target:
      kind: Deployment
      name: controller-manager
    patch: |-
      - op: replace
        path: /spec/template/spec/containers/0/resources/limits/memory
        value: 256Mi
      - op: replace
        path: /spec/template/spec/containers/0/resources/limits/cpu
        value: 1000m
```

3. Create `patches/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
spec:
  replicas: 2 # Increase replicas for HA
  template:
    spec:
      containers:
        - name: manager
          args:
            - --leader-elect
            - --poll-interval=30s # Custom poll interval
            - --log-level=info
          env:
            - name: POLL_INTERVAL
              value: "30s"
```

4. Apply the overlay:

```bash
kubectl apply -k .
```

### Method 4: Development Installation

For local development and testing:

```bash
# Clone the repository
git clone https://github.com/nusnewob/kube-changejob.git
cd kube-changejob

# Install CRDs
make install

# Run controller locally (outside cluster)
make run

# Or build and load into kind
kind create cluster --name kube-changejob-dev
make docker-build
kind load docker-image controller:latest --name kube-changejob-dev
make deploy
```

## Configuration

### Controller Configuration

The controller can be configured using command-line flags or environment variables.

#### Using Environment Variables

Edit the deployment to add environment variables:

```bash
kubectl edit deployment kube-changejob-controller-manager -n kube-changejob-system
```

Add environment variables:

```yaml
spec:
  template:
    spec:
      containers:
        - name: manager
          env:
            - name: POLL_INTERVAL
              value: "30s"
```

Or use kubectl:

```bash
kubectl set env deployment/kube-changejob-controller-manager \
  -n kube-changejob-system \
  POLL_INTERVAL=30s
```

#### Using Command-Line Flags

Patch the deployment to add command-line arguments:

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--poll-interval=30s"
  }]'
```

#### Available Configuration Options

| Flag                          | Environment Variable | Default   | Description                          |
| ----------------------------- | -------------------- | --------- | ------------------------------------ |
| `--poll-interval`             | `POLL_INTERVAL`      | `60s`     | How often to poll resources          |
| `--metrics-bind-address`      | -                    | `0`       | Metrics endpoint address             |
| `--health-probe-bind-address` | -                    | `:8081`   | Health probe address                 |
| `--leader-elect`              | -                    | `false`   | Enable leader election               |
| `--log-level`                 | -                    | `info`    | Log level (debug, info, warn, error) |
| `--log-format`                | -                    | `text`    | Log format (json or text)            |
| `--log-timestamp`             | -                    | `iso8601` | Timestamp format                     |

### High Availability Configuration

For production deployments, enable leader election and increase replicas:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-changejob-controller-manager
  namespace: kube-changejob-system
spec:
  replicas: 3 # Multiple replicas for HA
  template:
    spec:
      containers:
        - name: manager
          args:
            - --leader-elect # Enable leader election
            - --leader-elect-lease-duration=15s
            - --leader-elect-renew-deadline=10s
            - --leader-elect-retry-period=2s
```

Apply with:

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --patch '{"spec": {"replicas": 3}}'
```

### Resource Limits

Adjust resource limits based on your cluster size:

```yaml
spec:
  template:
    spec:
      containers:
        - name: manager
          resources:
            requests:
              cpu: 100m
              memory: 64Mi
            limits:
              cpu: 1000m
              memory: 256Mi
```

### Network Policies

Enable network policies for additional security:

```bash
# Apply network policies
kubectl apply -k github.com/nusnewob/kube-changejob/config/network-policy

# Or from local checkout
kubectl apply -k config/network-policy
```

### Prometheus Monitoring

Enable Prometheus metrics collection:

```bash
# Apply Prometheus ServiceMonitor
kubectl apply -k github.com/nusnewob/kube-changejob/config/prometheus

# Or from local checkout
kubectl apply -k config/prometheus
```

## Verification

### Check Installation Status

```bash
# Check namespace
kubectl get namespace kube-changejob-system

# Check deployment
kubectl get deployment -n kube-changejob-system

# Check pods
kubectl get pods -n kube-changejob-system

# Check CRD
kubectl get crd changetriggeredjobs.triggers.changejob.dev

# Check webhook configurations
kubectl get validatingwebhookconfiguration
kubectl get mutatingwebhookconfiguration
```

### Verify Controller is Running

```bash
# Check pod status
kubectl get pods -n kube-changejob-system

# Check logs
kubectl logs -n kube-changejob-system \
  deployment/kube-changejob-controller-manager

# Check health endpoints
kubectl port-forward -n kube-changejob-system \
  deployment/kube-changejob-controller-manager 8081:8081 &

curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
```

### Verify Webhooks

```bash
# Check webhook certificates
kubectl get certificate -n kube-changejob-system

# Check webhook endpoints
kubectl get validatingwebhookconfiguration \
  kube-changejob-validating-webhook-configuration -o yaml

kubectl get mutatingwebhookconfiguration \
  kube-changejob-mutating-webhook-configuration -o yaml
```

### Test Basic Functionality

Create a test ChangeTriggeredJob:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: test-trigger
  namespace: default
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: test
            image: busybox:latest
            command: ["echo", "test"]
          restartPolicy: Never
  resources:
  - apiVersion: v1
    kind: ConfigMap
    name: test-config
    namespace: default
EOF

# Create the ConfigMap
kubectl create configmap test-config --from-literal=key=value

# Check ChangeTriggeredJob status
kubectl get changetriggeredjob test-trigger

# Clean up
kubectl delete changetriggeredjob test-trigger
kubectl delete configmap test-config
```

## Upgrading

### Upgrading from Previous Version

```bash
# Using kubectl
kubectl apply -k github.com/nusnewob/kube-changejob/config/default

# Or from local checkout
git pull origin main
make deploy
```

### Rolling Back

If you encounter issues after upgrading:

```bash
# View deployment history
kubectl rollout history deployment/kube-changejob-controller-manager \
  -n kube-changejob-system

# Roll back to previous version
kubectl rollout undo deployment/kube-changejob-controller-manager \
  -n kube-changejob-system

# Or roll back to specific revision
kubectl rollout undo deployment/kube-changejob-controller-manager \
  -n kube-changejob-system --to-revision=2
```

## Uninstallation

### Complete Removal

```bash
# Delete all ChangeTriggeredJob resources first
kubectl delete changetriggeredjobs --all --all-namespaces

# Uninstall kube-changejob
kubectl delete -k github.com/nusnewob/kube-changejob/config/default

# Or from local checkout
make undeploy

# Optionally remove cert-manager (if not used by other applications)
# kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml
```

### Keeping CRD (Preserve Resources)

If you want to keep ChangeTriggeredJob resources:

```bash
# Delete controller only
kubectl delete deployment kube-changejob-controller-manager -n kube-changejob-system
kubectl delete service kube-changejob-controller-manager-metrics-service -n kube-changejob-system
kubectl delete service kube-changejob-webhook-service -n kube-changejob-system

# Keep CRD and resources
# kubectl get changetriggeredjobs --all-namespaces
```

## Troubleshooting Installation

### cert-manager Issues

```bash
# Check cert-manager pods
kubectl get pods -n cert-manager

# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager

# Verify certificates are issued
kubectl get certificate -n kube-changejob-system
kubectl describe certificate -n kube-changejob-system
```

### Webhook Issues

```bash
# Check webhook service
kubectl get service -n kube-changejob-system

# Check webhook endpoints
kubectl get endpoints -n kube-changejob-system

# Test webhook connectivity
kubectl run test-pod --rm -it --restart=Never \
  --image=curlimages/curl -- \
  curl -k https://kube-changejob-webhook-service.kube-changejob-system.svc:443
```

### Controller Not Starting

```bash
# Check pod status
kubectl describe pod -n kube-changejob-system \
  -l control-plane=controller-manager

# Check logs
kubectl logs -n kube-changejob-system \
  -l control-plane=controller-manager --all-containers=true

# Check events
kubectl get events -n kube-changejob-system --sort-by='.lastTimestamp'
```

### RBAC Issues

```bash
# Check service account
kubectl get serviceaccount -n kube-changejob-system

# Check cluster role
kubectl get clusterrole kube-changejob-manager-role

# Check cluster role binding
kubectl get clusterrolebinding kube-changejob-manager-rolebinding

# Verify permissions
kubectl auth can-i --list --as=system:serviceaccount:kube-changejob-system:kube-changejob-controller-manager
```

## Next Steps

- Read the [User Guide](user-guide) to learn how to use kube-changejob
- Check the [API Reference](api-reference) for detailed specifications
- Review [Configuration](configuration) for advanced settings
- Explore [Examples](examples) for common use cases

## Getting Help

If you encounter issues during installation:

1. Check the [Troubleshooting](#troubleshooting-installation) section above
2. Review controller logs: `kubectl logs -n kube-changejob-system deployment/kube-changejob-controller-manager`
3. Check [GitHub Issues](https://github.com/nusnewob/kube-changejob/issues)
4. Open a new issue with installation details
