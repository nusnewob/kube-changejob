---
layout: default
title: User Guide
nav_order: 3
description: "Complete usage guide for kube-changejob"
---

# User Guide

This guide provides comprehensive instructions for using kube-changejob to automate workflows based on Kubernetes resource changes.

## Table of Contents

- [Introduction](#introduction)
- [Getting Started](#getting-started)
- [Basic Usage](#basic-usage)
- [Advanced Usage](#advanced-usage)
- [Monitoring and Debugging](#monitoring-and-debugging)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Introduction

kube-changejob is a Kubernetes operator that automatically triggers jobs when specified resources change. It continuously monitors resources and creates jobs based on your defined templates when changes are detected.

### Key Concepts

- **ChangeTriggeredJob (CTJ)**: The custom resource that defines what to watch and what to run
- **Resource Watching**: Monitoring Kubernetes resources for changes
- **Trigger Conditions**: Rules that determine when to create jobs
- **Job Template**: The specification for jobs to create when triggered
- **Cooldown Period**: Minimum time between job triggers
- **History Management**: Automatic cleanup of old jobs

## Getting Started

### Prerequisites

Before using kube-changejob, ensure you have:

1. A Kubernetes cluster (v1.29 or later)
2. kubectl configured to access your cluster
3. Appropriate RBAC permissions to create ChangeTriggeredJobs
4. cert-manager installed (for webhooks)

### Installing kube-changejob

See the [Installation Guide](installation) for detailed installation instructions.

Quick install:

```bash
kubectl apply -k github.com/nusnewob/kube-changejob/config/default
```

### Verifying Installation

Check that the controller is running:

```bash
kubectl get pods -n kube-changejob-system
kubectl get crd changetriggeredjobs.triggers.changejob.dev
```

## Basic Usage

### Creating Your First ChangeTriggeredJob

Let's create a simple ChangeTriggeredJob that triggers when a ConfigMap changes:

1. Create a ConfigMap to watch:

```bash
kubectl create configmap my-config --from-literal=key=value
```

2. Create a ChangeTriggeredJob:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: my-first-trigger
  namespace: default
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: hello
              image: busybox:latest
              command: ["sh", "-c", "echo 'ConfigMap changed at:' $(date)"]
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: my-config
      namespace: default
```

3. Apply the configuration:

```bash
kubectl apply -f changetriggeredjob.yaml
```

4. Update the ConfigMap to trigger the job:

```bash
kubectl patch configmap my-config -p '{"data":{"key":"newvalue"}}'
```

5. Wait for the cooldown period (default 60s), then check for created jobs:

```bash
kubectl get jobs -l changejob.dev/owner=my-first-trigger
```

### Viewing Status

Check the status of your ChangeTriggeredJob:

```bash
# Get basic info
kubectl get changetriggeredjobs

# Get detailed status
kubectl describe changetriggeredjob my-first-trigger

# View status in YAML format
kubectl get changetriggeredjob my-first-trigger -o yaml
```

Key status fields to check:

- `status.lastTriggeredTime`: When the last job was created
- `status.lastJobName`: Name of the most recent job
- `status.lastJobStatus`: Status of the last job (Active/Succeeded/Failed)
- `status.conditions`: Current state conditions

### Viewing Created Jobs

List jobs created by a ChangeTriggeredJob:

```bash
# List all jobs with the owner label
kubectl get jobs -l changejob.dev/owner=my-first-trigger

# View job logs
kubectl logs job/<job-name>

# Describe a specific job
kubectl describe job <job-name>
```

## Advanced Usage

### Watching Multiple Resources

You can watch multiple resources and control when to trigger:

#### Trigger on Any Change (OR Logic)

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: multi-resource-any
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: sync
              image: my-sync-tool:latest
              command: ["sync"]
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
  condition: Any # Trigger if ConfigMap OR Secret changes
  cooldown: 120s
```

#### Trigger on All Changes (AND Logic)

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: multi-resource-all
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: sync
              image: my-sync-tool:latest
              command: ["sync", "--coordinated"]
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
  condition: All # Trigger only if BOTH ConfigMap AND Secret change
  cooldown: 300s
```

### Watching Specific Fields

Instead of watching the entire resource, you can monitor specific fields:

#### Watch Deployment Image Changes

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: image-watcher
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: notify
              image: notification-tool:latest
              env:
                - name: WEBHOOK_URL
                  value: "https://hooks.slack.com/..."
          restartPolicy: Never
  resources:
    - apiVersion: apps/v1
      kind: Deployment
      name: web-app
      namespace: default
      fields:
        - "spec.template.spec.containers[*].image"
  cooldown: 30s
```

#### Watch ConfigMap Data Only

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: config-data-watcher
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: reload
              image: reload-tool:latest
              command: ["reload-config"]
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: default
      fields:
        - "data" # Only watch the data field
```

#### Watch Multiple Fields

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: multi-field-watcher
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: check
              image: checker:latest
          restartPolicy: Never
  resources:
    - apiVersion: apps/v1
      kind: Deployment
      name: app
      namespace: default
      fields:
        - "spec.replicas"
        - "spec.template.spec.containers[*].image"
        - "spec.template.spec.containers[*].resources"
```

### Watching Cluster-Scoped Resources

You can watch cluster-scoped resources like Nodes, ClusterRoles, etc.:

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
          serviceAccountName: cluster-reader
          containers:
            - name: alert
              image: alert-tool:latest
              command: ["alert", "Node changed"]
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: Node
      name: worker-1
      # No namespace field for cluster-scoped resources
  cooldown: 300s
```

### Customizing Job Templates

#### Job with Environment Variables

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: processor
              image: processor:latest
              env:
                - name: CONFIG_NAME
                  value: "app-config"
                - name: LOG_LEVEL
                  value: "info"
          restartPolicy: Never
```

#### Job with ConfigMap/Secret Mounts

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: processor
              image: processor:latest
              volumeMounts:
                - name: config
                  mountPath: /config
                - name: secret
                  mountPath: /secret
                  readOnly: true
          volumes:
            - name: config
              configMap:
                name: app-config
            - name: secret
              secret:
                secretName: app-secret
          restartPolicy: Never
```

#### Job with Resource Limits

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: heavy-processor
              image: processor:latest
              resources:
                requests:
                  memory: "256Mi"
                  cpu: "500m"
                limits:
                  memory: "512Mi"
                  cpu: "1000m"
          restartPolicy: Never
```

#### Job with Service Account

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: my-job-sa
          containers:
            - name: k8s-client
              image: k8s-client:latest
              command: ["kubectl", "get", "pods"]
          restartPolicy: Never
```

### Adjusting Cooldown Period

Control how often jobs can be triggered:

```yaml
spec:
  # Short cooldown for frequent updates
  cooldown: 30s

  # Or longer cooldown for expensive operations
  # cooldown: 15m

  # Or very long cooldown
  # cooldown: 1h
```

### Managing Job History

Configure how many historical jobs to keep:

```yaml
spec:
  # Keep more jobs for debugging
  history: 10

  # Or keep fewer to save resources
  # history: 3

  # Minimum is 1
  # history: 1
```

## Real-World Use Cases

### 1. Configuration Synchronization

Trigger sync jobs when configuration changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: config-sync
  namespace: production
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: sync
              image: config-sync-tool:v1.0
              command: ["sync-config"]
              env:
                - name: TARGET_SYSTEMS
                  value: "system1,system2,system3"
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: production
  cooldown: 300s
  history: 5
```

### 2. Deployment Notifications

Send notifications when deployments change:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: deployment-notifier
  namespace: production
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: notify
              image: notification-service:latest
              env:
                - name: SLACK_WEBHOOK
                  valueFrom:
                    secretKeyRef:
                      name: slack-webhook
                      key: url
                - name: MESSAGE
                  value: "Production deployment updated"
          restartPolicy: Never
  resources:
    - apiVersion: apps/v1
      kind: Deployment
      name: web-app
      namespace: production
      fields:
        - "spec.template.spec.containers[*].image"
  cooldown: 60s
```

### 3. Backup Automation

Trigger backups when data changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: backup-trigger
  namespace: database
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: backup-tool:latest
              command: ["backup", "--incremental"]
              volumeMounts:
                - name: backup-storage
                  mountPath: /backups
          volumes:
            - name: backup-storage
              persistentVolumeClaim:
                claimName: backup-pvc
          restartPolicy: OnFailure
      backoffLimit: 3
  resources:
    - apiVersion: v1
      kind: Secret
      name: database-credentials
      namespace: database
    - apiVersion: v1
      kind: ConfigMap
      name: database-config
      namespace: database
  condition: Any
  cooldown: 3600s # 1 hour
  history: 24 # Keep 24 hours of backups
```

### 4. Validation Pipeline

Run validation when resources are updated:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: policy-validator
  namespace: compliance
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: validator-sa
          containers:
            - name: validate
              image: policy-validator:latest
              command: ["validate-policies"]
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: security-policies
      namespace: compliance
  cooldown: 120s
```

## Monitoring and Debugging

### Checking ChangeTriggeredJob Status

```bash
# List all ChangeTriggeredJobs
kubectl get ctj -A

# Get detailed status
kubectl describe ctj my-trigger

# Watch for changes
kubectl get ctj my-trigger -w

# View full status
kubectl get ctj my-trigger -o jsonpath='{.status}' | jq
```

### Monitoring Created Jobs

```bash
# List jobs for a specific ChangeTriggeredJob
kubectl get jobs -l changejob.dev/owner=my-trigger

# Watch job creation
kubectl get jobs -l changejob.dev/owner=my-trigger -w

# View job status
kubectl describe job <job-name>

# Get job logs
kubectl logs job/<job-name>

# Follow logs in real-time
kubectl logs -f job/<job-name>
```

### Checking Resource Hashes

View the current hash state of watched resources:

```bash
kubectl get ctj my-trigger -o jsonpath='{.status.resourceHashes}' | jq
```

### Viewing Controller Logs

Debug controller behavior:

```bash
# View controller logs
kubectl logs -n kube-changejob-system \
  deployment/kube-changejob-controller-manager

# Follow logs
kubectl logs -f -n kube-changejob-system \
  deployment/kube-changejob-controller-manager

# Filter for specific ChangeTriggeredJob
kubectl logs -n kube-changejob-system \
  deployment/kube-changejob-controller-manager | \
  grep "my-trigger"
```

### Checking Conditions

Understand the current state through conditions:

```bash
kubectl get ctj my-trigger -o jsonpath='{.status.conditions}' | jq
```

Common condition types:

- `Available`: Is the CTJ functioning correctly?
- `Progressing`: Is work in progress?
- `Degraded`: Are there any issues?

## Best Practices

### 1. Use Appropriate Cooldown Periods

- **Short cooldown (30s-60s)**: For frequently changing resources where immediate response is needed
- **Medium cooldown (5m-15m)**: For most use cases, balances responsiveness and resource usage
- **Long cooldown (1h+)**: For expensive operations or batch processing

### 2. Watch Specific Fields

Instead of watching entire resources, monitor only relevant fields:

```yaml
# Good: Watch only what matters
fields:
  - "spec.template.spec.containers[*].image"
# Avoid: Watching everything when you only care about specific fields
# (omitting fields or using ["*"] watches everything)
```

### 3. Use Meaningful Names

Choose descriptive names for ChangeTriggeredJobs:

```yaml
# Good
name: nginx-config-sync
name: database-backup-trigger
name: deployment-image-notifier

# Avoid
name: trigger1
name: test
name: my-ctj
```

### 4. Set Appropriate History Limits

Balance debugging needs with resource usage:

```yaml
# Development/debugging
history: 10

# Production (most cases)
history: 5

# High-volume triggers
history: 3
```

### 5. Use Resource Limits in Job Templates

Prevent runaway jobs:

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: processor
              resources:
                limits:
                  memory: "512Mi"
                  cpu: "500m"
```

### 6. Implement Proper Error Handling

Use appropriate restart policies:

```yaml
spec:
  jobTemplate:
    spec:
      backoffLimit: 3 # Retry failed jobs
      template:
        spec:
          containers:
            - name: processor
              # ...
          restartPolicy: OnFailure # Retry on failure
```

### 7. Use Labels and Annotations

Organize and document your ChangeTriggeredJobs:

```yaml
metadata:
  name: my-trigger
  labels:
    app: myapp
    environment: production
    team: platform
  annotations:
    description: "Syncs configuration to external systems"
    owner: "platform-team@example.com"
    runbook: "https://wiki.example.com/runbooks/config-sync"
```

### 8. Monitor Job Success Rates

Regularly check job statuses:

```bash
# Check for failed jobs
kubectl get jobs -l changejob.dev/owner=my-trigger --field-selector status.successful=0

# Monitor success rate
kubectl get ctj my-trigger -o jsonpath='{.status.lastJobStatus}'
```

### 9. Test in Non-Production First

Always test new ChangeTriggeredJobs in development:

```yaml
# Development
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: config-sync-dev
  namespace: development
spec:
  cooldown: 30s  # Shorter cooldown for testing
  # ...

# Production (deploy after testing)
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: config-sync-prod
  namespace: production
spec:
  cooldown: 300s  # Longer cooldown for stability
  # ...
```

### 10. Document Your Triggers

Use annotations to document purpose and behavior:

```yaml
metadata:
  annotations:
    description: |
      Triggers backup jobs when database credentials change.
      Backs up data to S3 with encryption enabled.
    trigger-frequency: "Expected: 1-2 times per week"
    on-call: "database-team@example.com"
```

## Troubleshooting

### Jobs Not Being Created

**Problem**: ChangeTriggeredJob exists but no jobs are created when resources change.

**Solutions**:

1. Check if cooldown period has elapsed:

```bash
kubectl get ctj my-trigger -o jsonpath='{.status.lastTriggeredTime}'
```

2. Verify resources are actually changing:

```bash
# Check current hashes
kubectl get ctj my-trigger -o jsonpath='{.status.resourceHashes}' | jq

# Force a change
kubectl annotate configmap my-config test=value-$(date +%s)
```

3. Check controller logs:

```bash
kubectl logs -n kube-changejob-system deployment/kube-changejob-controller-manager
```

4. Verify resource exists and is accessible:

```bash
kubectl get configmap my-config -n default
```

### Jobs Failing Immediately

**Problem**: Created jobs fail immediately or repeatedly.

**Solutions**:

1. Check job logs:

```bash
kubectl logs job/<job-name>
```

2. Check job events:

```bash
kubectl describe job <job-name>
```

3. Verify image exists and is pullable:

```bash
kubectl run test --image=<your-image> --rm -it --restart=Never -- echo "test"
```

4. Check resource limits:

```yaml
# Add or adjust limits
resources:
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### "All" Condition Not Triggering

**Problem**: Using `condition: All` but jobs never trigger.

**Solutions**:

1. Verify all resources have changed since last trigger:

```bash
kubectl get ctj my-trigger -o jsonpath='{.status.resourceHashes}' | jq
```

2. Check each resource individually:

```bash
kubectl get configmap my-config
kubectl get secret my-secret
```

3. After all resources are changed, wait for the next poll interval (default 60s)

### Webhook Validation Errors

**Problem**: Getting errors when creating/updating ChangeTriggeredJob.

**Common Errors and Solutions**:

```
Error: resource kind does not exist
```

Solution: Check that the resource kind is valid and properly capitalized:

```yaml
# Correct
kind: ConfigMap

# Incorrect
kind: configmap
```

```
Error: namespace is required for namespaced resources
```

Solution: Add namespace for namespaced resources:

```yaml
resources:
  - apiVersion: v1
    kind: ConfigMap
    name: my-config
    namespace: default # Add this
```

```
Error: namespace must not be set for cluster-scoped resources
```

Solution: Remove namespace for cluster-scoped resources:

```yaml
resources:
  - apiVersion: v1
    kind: Node
    name: worker-1
    # Remove namespace field
```

### Too Many Jobs Being Created

**Problem**: Jobs are created too frequently.

**Solutions**:

1. Increase cooldown period:

```yaml
spec:
  cooldown: 300s # Increase from default 60s
```

2. Use "All" condition instead of "Any":

```yaml
spec:
  condition: All # Require all resources to change
```

3. Watch specific fields instead of entire resources:

```yaml
spec:
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: my-config
      fields:
        - "data.important-key" # Only watch this field
```

### Permission Errors

**Problem**: Jobs fail with permission errors or ChangeTriggeredJob can't watch resources.

**Solutions**:

1. For job execution, add appropriate service account:

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: my-job-sa
```

2. For controller watching resources, check controller RBAC:

```bash
kubectl get clusterrole kube-changejob-manager-role -o yaml
```

3. For user creating ChangeTriggeredJobs, verify permissions:

```bash
kubectl auth can-i create changetriggeredjobs
```

## Next Steps

- Review the [API Reference](api-reference) for detailed specification
- Check [Examples](examples) for more use cases
- Learn about [Configuration](configuration) options
- Read the [Installation Guide](installation) for deployment options
- Learn the [Release Process](release) to create and manage releases

## Getting Help

If you encounter issues:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review controller logs
3. Check [GitHub Issues](https://github.com/nusnewob/kube-changejob/issues)
4. Open a new issue with details about your problem
