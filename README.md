# kube-changejob

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/nusnewob/kube-changejob)](https://goreportcard.com/report/github.com/nusnewob/kube-changejob)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/nusnewob/kube-changejob?include_prereleases)](https://github.com/nusnewob/kube-changejob/releases)
[![Docker Image](https://img.shields.io/badge/container-ghcr.io-blue)](https://ghcr.io/nusnewob/kube-changejob)
[![Helm Chart](https://img.shields.io/badge/helm-chart-blue)](https://ghcr.io/nusnewob/charts/kube-changejob)
[![GitHub branch check runs](https://img.shields.io/github/check-runs/nusnewob/kube-changejob/main)](https://github.com/nusnewob/kube-changejob/actions?query=branch%3Amain)
[![Codecov](https://img.shields.io/codecov/c/github/nusnewob/kube-changejob?component=controller)](https://app.codecov.io/gh/nusnewob/kube-changejob)

A Kubernetes operator that automatically triggers jobs when watched Kubernetes resources change.

## Overview

kube-changejob monitors specified Kubernetes resources and triggers jobs when changes are detected. It provides a declarative way to automate workflows in response to resource modifications, making it ideal for:

- Configuration synchronization workflows
- Automated deployment pipelines
- Resource validation and compliance checks
- Backup and snapshot operations
- Custom event-driven automation

## Features

- **Flexible Resource Watching**: Monitor any Kubernetes resource (ConfigMaps, Secrets, Deployments, etc.)
- **Field-Specific Monitoring**: Watch entire resources or specific fields using JSONPath
- **Trigger Conditions**: Configure "Any" or "All" logic for multi-resource triggers
- **Cooldown Period**: Prevent excessive job creation with configurable cooldown
- **Job History Management**: Automatically clean up old jobs with history limits
- **Webhook Validation**: Built-in validation and defaulting webhooks
- **High Availability**: Supports leader election for HA deployments
- **Secure by Default**: TLS-enabled webhooks and metrics, restrictive pod security

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.29+)
- kubectl configured to access your cluster
- cert-manager (for webhook certificates)

### Installation

#### Option 1: Using kubectl (Stable Release)

```bash
# Install cert-manager (if not already installed)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml

# Install kube-changejob (replace VERSION with desired version)
kubectl apply -f https://github.com/nusnewob/kube-changejob/releases/latest/download/install.yaml

# Verify the installation
kubectl get pods -n kube-changejob-system
```

#### Option 2: Using Helm

```bash
# Install from OCI registry
helm install kube-changejob oci://ghcr.io/nusnewob/charts/kube-changejob --version 0.1.0

# Or with custom values
helm install kube-changejob oci://ghcr.io/nusnewob/charts/kube-changejob \
  --version 0.1.0 \
  --set image.tag=v0.1.0
```

#### Option 3: Using Kustomize (Development)

```bash
kubectl apply -k github.com/nusnewob/kube-changejob/config/default
```

### Basic Usage

Create a ChangeTriggeredJob that triggers when a ConfigMap changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: config-watcher
  namespace: default
spec:
  # Job template - what to run when triggered
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: runner
              image: busybox:latest
              command: ["sh", "-c", "echo 'ConfigMap changed at:' $(date)"]
          restartPolicy: Never

  # Resources to watch
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: my-config
      namespace: default

  # Trigger when any resource changes (default)
  condition: Any

  # Wait 60 seconds between triggers (default)
  cooldown: 60s

  # Keep last 5 jobs (default)
  history: 5
```

Apply the configuration:

```bash
kubectl apply -f changejob.yaml
```

Monitor the status:

```bash
kubectl get changetriggeredjobs config-watcher -o yaml
```

## Documentation

Comprehensive documentation is available in the [docs/](./docs) directory:

- [Installation Guide](./docs/installation.md) - Detailed installation instructions
- [User Guide](./docs/user-guide.md) - Complete usage guide with examples
- [API Reference](./docs/api-reference.md) - CRD specification and API details
- [Configuration](./docs/configuration.md) - Controller configuration options
- [Examples](./docs/examples.md) - Real-world usage examples
- [Release Process](./docs/release.md) - How to create and manage releases

## How It Works

1. **Resource Polling**: The controller periodically polls watched resources (default: every 60 seconds)
2. **Change Detection**: Computes SHA256 hashes of resource data to detect changes
3. **Trigger Evaluation**: Evaluates trigger conditions (Any/All) and cooldown period
4. **Job Creation**: Creates a new Job from the jobTemplate when triggered
5. **History Management**: Automatically cleans up old jobs based on history limit

## Configuration

The controller can be configured using command-line flags or environment variables:

| Flag                          | Environment Variable | Default   | Description                                                                  |
| ----------------------------- | -------------------- | --------- | ---------------------------------------------------------------------------- |
| `--poll-interval`             | `POLL_INTERVAL`      | `60s`     | How often to poll resources                                                  |
| `--metrics-bind-address`      | -                    | `0`       | Metrics endpoint address                                                     |
| `--health-probe-bind-address` | -                    | `:8081`   | Health probe address                                                         |
| `--leader-elect`              | -                    | `false`   | Enable leader election                                                       |
| `--log-level`                 | -                    | `info`    | Log level (debug, info, warn, error)                                         |
| `--log-format`                | -                    | `text`    | Log format (json or text)                                                    |
| `--log-timestamp`             | -                    | `rfc3339` | Log timestamp formart (epoch, millis, nano, iso8601, rfc3339 or rfc3339nano) |
| `--debug`                     | -                    | `false`   | Enable debug info                                                            |

Example with custom poll interval:

```bash
# Via environment variable
kubectl set env deployment/kube-changejob-controller-manager \
  -n kube-changejob-system \
  POLL_INTERVAL=30s

# Via command-line flag
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--poll-interval=30s"}]'
```

## Examples

### Watch Multiple Resources with "All" Condition

Trigger only when all specified resources have changed:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: multi-resource-watcher
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: sync
              image: my-sync-tool:latest
              command: ["sync-all"]
          restartPolicy: Never

  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: default
    - apiVersion: v1
      kind: Secret
      name: app-secret
      namespace: default
    - apiVersion: apps/v1
      kind: Deployment
      name: app
      namespace: default

  condition: All # Trigger only when all three resources change
  cooldown: 300s # 5-minute cooldown
```

### Watch Specific Fields

Monitor only specific fields using JSONPath:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: deployment-image-watcher
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: notify
              image: notification-tool:latest
              command: ["notify", "Deployment image updated"]
          restartPolicy: Never

  resources:
    - apiVersion: apps/v1
      kind: Deployment
      name: my-app
      namespace: default
      fields:
        - "spec.template.spec.containers[*].image" # Watch only container images

  cooldown: 30s
```

### Watch Cluster-Scoped Resources

Monitor cluster-wide resources:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: node-watcher
  namespace: kube-changejob-system
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: alert
              image: alert-tool:latest
              command: ["alert", "Node changes detected"]
          restartPolicy: Never

  resources:
    - apiVersion: v1
      kind: Node
      name: worker-1
      # No namespace - Node is cluster-scoped

  cooldown: 120s
```

## Development

### Prerequisites

- Go 1.24+
- Docker
- kubectl
- kind (for local testing)
- kubebuilder 4.10.1+

### Building from Source

```bash
# Clone the repository
git clone https://github.com/nusnewob/kube-changejob.git
cd kube-changejob

# Build the controller
make build

# Run tests
make test

# Run end-to-end tests
make test-e2e
```

### Local Development

```bash
# Install CRDs
make install

# Run controller locally
make run

# In another terminal, apply test resources
kubectl apply -f config/samples/
```

### Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    ChangeTriggeredJob                       │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Spec:                                                 │  │
│  │  - jobTemplate: Job template to create                │  │
│  │  - resources: List of resources to watch              │  │
│  │  - condition: "Any" or "All"                          │  │
│  │  - cooldown: Minimum time between triggers            │  │
│  │  - history: Number of jobs to keep                    │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              ChangeTriggeredJob Controller                  │
│                                                             │
│  1. Poll watched resources (every 60s)                      │
│  2. Compute SHA256 hash of resource data                    │
│  3. Compare with stored hashes                              │
│  4. Evaluate trigger condition (Any/All)                    │
│  5. Check cooldown period                                   │
│  6. Create Job from template if triggered                   │
│  7. Update status with job info                             │
│  8. Clean up old jobs (history limit)                       │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Jobs                          │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │ Job 1    │  │ Job 2    │  │ Job 3    │  │ Job 4    │     │
│  │ (oldest) │  │          │  │          │  │ (latest) │     │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘     │
│                                                             │
│  Jobs owned by ChangeTriggeredJob (automatic cleanup)       │
└─────────────────────────────────────────────────────────────┘
```

## Monitoring

### Metrics

The controller exposes Prometheus metrics on port 8443 (HTTPS) or 8080 (HTTP):

- Standard controller-runtime metrics
- Custom resource metrics

To enable Prometheus monitoring:

```bash
kubectl apply -k config/prometheus
```

### Health Checks

Health and readiness probes are available on port 8081:

- `/healthz` - Health check
- `/readyz` - Readiness check

## Security

### Reporting Security Issues

Please see [SECURITY.md](./SECURITY.md) for information on reporting security vulnerabilities.

### Security Features

- TLS-enabled webhooks and metrics endpoints
- Restrictive pod security context (non-root, read-only filesystem)
- RBAC with minimal required permissions
- Network policies for traffic control
- Webhook validation for resource specifications

## Troubleshooting

### Jobs Not Triggering

1. Check controller logs:

```bash
kubectl logs -n kube-changejob-system deployment/kube-changejob-controller-manager
```

2. Verify ChangeTriggeredJob status:

```bash
kubectl describe changetriggeredjob <name>
```

3. Check if cooldown period has elapsed
4. Ensure watched resources exist and are accessible

### Webhook Errors

1. Verify cert-manager is running:

```bash
kubectl get pods -n cert-manager
```

2. Check webhook certificates:

```bash
kubectl get certificate -n kube-changejob-system
```

3. Check webhook configuration:

```bash
kubectl get validatingwebhookconfiguration
kubectl get mutatingwebhookconfiguration
```

### Permission Issues

1. Verify RBAC permissions:

```bash
kubectl get clusterrole kube-changejob-manager-role -o yaml
```

2. Check service account:

```bash
kubectl get serviceaccount -n kube-changejob-system
```

## Uninstallation

```bash
# Delete all ChangeTriggeredJob resources
kubectl delete changetriggeredjobs --all --all-namespaces

# Uninstall the operator
kubectl delete -k github.com/nusnewob/kube-changejob/config/default
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:

- [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) - Kubernetes operator framework
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Controller runtime library
- [cert-manager](https://cert-manager.io/) - Certificate management for webhooks

## Contact

- **Author**: Bowen Sun
- **Repository**: [github.com/nusnewob/kube-changejob](https://github.com/nusnewob/kube-changejob)
- **Issues**: [GitHub Issues](https://github.com/nusnewob/kube-changejob/issues)
