---
layout: default
title: Home
nav_order: 1
description: "kube-changejob - Kubernetes operator for resource change triggered jobs"
permalink: /
---

# kube-changejob

A Kubernetes operator that automatically triggers jobs when watched Kubernetes resources change.

## Overview

kube-changejob monitors specified Kubernetes resources and triggers jobs when changes are detected. It provides a declarative way to automate workflows in response to resource modifications.

## Key Features

- **Flexible Resource Watching**: Monitor any Kubernetes resource (ConfigMaps, Secrets, Deployments, etc.)
- **Field-Specific Monitoring**: Watch entire resources or specific fields using JSONPath
- **Trigger Conditions**: Configure "Any" or "All" logic for multi-resource triggers
- **Cooldown Period**: Prevent excessive job creation with configurable cooldown
- **Job History Management**: Automatically clean up old jobs with history limits
- **Webhook Validation**: Built-in validation and defaulting webhooks
- **High Availability**: Supports leader election for HA deployments
- **Secure by Default**: TLS-enabled webhooks and metrics, restrictive pod security

## Quick Start

### Installation

```bash
# Install cert-manager (if not already installed)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml

# Install kube-changejob
kubectl apply -k github.com/nusnewob/kube-changejob/config/default
```

### Basic Example

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

## Use Cases

kube-changejob is ideal for:

- **Configuration Synchronization**: Sync configs to external systems when they change
- **Automated Deployment Pipelines**: Trigger workflows on deployment updates
- **Resource Validation**: Run compliance checks when resources are modified
- **Backup Operations**: Trigger backups when data or credentials change
- **Event-Driven Automation**: Create custom workflows triggered by resource changes

## Documentation

- [**Installation Guide**](installation) - Detailed installation instructions
- [**User Guide**](user-guide) - Complete usage guide with examples
- [**API Reference**](api-reference) - CRD specification and API details
- [**Configuration**](configuration) - Controller configuration options
- [**Examples**](examples) - Real-world usage examples

## How It Works

1. **Resource Polling**: The controller periodically polls watched resources (default: every 60 seconds)
2. **Change Detection**: Computes SHA256 hashes of resource data to detect changes
3. **Trigger Evaluation**: Evaluates trigger conditions (Any/All) and cooldown period
4. **Job Creation**: Creates a new Job from the jobTemplate when triggered
5. **History Management**: Automatically cleans up old jobs based on history limit

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
│  - Poll watched resources                                   │
│  - Compute SHA256 hashes                                    │
│  - Compare with stored hashes                               │
│  - Evaluate trigger condition                               │
│  - Create Job from template                                 │
│  - Update status and clean up old jobs                      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Jobs                          │
│  Jobs owned by ChangeTriggeredJob (automatic cleanup)       │
└─────────────────────────────────────────────────────────────┘
```

## Community

- **Repository**: [github.com/nusnewob/kube-changejob](https://github.com/nusnewob/kube-changejob)
- **Issues**: [GitHub Issues](https://github.com/nusnewob/kube-changejob/issues)
- **License**: Apache License 2.0

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](https://github.com/nusnewob/kube-changejob/blob/main/CONTRIBUTING.md) for guidelines.

## Security

For information on reporting security vulnerabilities, see [SECURITY.md](https://github.com/nusnewob/kube-changejob/blob/main/SECURITY.md).
