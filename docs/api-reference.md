---
layout: default
title: API Reference
nav_order: 4
description: "CRD specification and API documentation"
---

# API Reference

This document provides detailed information about the ChangeTriggeredJob Custom Resource Definition (CRD).

## Table of Contents

- [Overview](#overview)
- [Resource Definition](#resource-definition)
- [Spec Fields](#spec-fields)
- [Status Fields](#status-fields)
- [Types Reference](#types-reference)
- [Examples](#examples)

## Overview

**API Group**: `triggers.changejob.dev`  
**API Version**: `v1alpha`  
**Kind**: `ChangeTriggeredJob`  
**Scope**: Namespaced  
**Short Names**: `ctj`, `ctjs`

The ChangeTriggeredJob resource defines a job that is automatically triggered when specified Kubernetes resources change.

## Resource Definition

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: string # Required: Name of the ChangeTriggeredJob
  namespace: string # Required: Namespace (namespaced resource)
  labels: {} # Optional: Labels
  annotations: {} # Optional: Annotations
spec: # Required: ChangeTriggeredJobSpec
  jobTemplate: {} # Required: Job template
  resources: [] # Required: List of resources to watch
  condition: string # Optional: "Any" or "All" (default: "Any")
  cooldown: duration # Optional: Cooldown period (default: 60s)
  history: int32 # Optional: Job history limit (default: 5)
status: # Managed by controller
  conditions: [] # Status conditions
  resourceHashes: [] # Resource state hashes
  lastTriggeredTime: time # Last trigger timestamp
  lastJobName: string # Last created job name
  lastJobStatus: string # Last job status
```

## Spec Fields

### `jobTemplate` (required)

Type: `batchv1.JobTemplateSpec`

The Job template defines the Job to create when the trigger condition is met. This follows the standard Kubernetes Job template specification.

**Structure**:

```yaml
spec:
  jobTemplate:
    metadata: # Optional: Job metadata
      labels: {} # Labels for created jobs
      annotations: {} # Annotations for created jobs
    spec: # Required: Job spec
      template: # Required: Pod template
        spec: # Required: Pod spec
          containers: # Required: Container list
            - name: string
              image: string
              command: []
              args: []
          restartPolicy: Never|OnFailure # Required
      backoffLimit: int32 # Optional: Job backoff limit
      completions: int32 # Optional: Desired completions
      parallelism: int32 # Optional: Parallel pods
```

**Notes**:

- The `metadata.name` field in the Job template should not be set; use `metadata.generateName` instead or leave it empty
- Created jobs will automatically have:
  - Label: `changejob.dev/owner=<changetriggeredjob-name>`
  - Owner reference to the ChangeTriggeredJob
  - Unique generated name based on the ChangeTriggeredJob name

**Example**:

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: sync
              image: busybox:latest
              command: ["sh", "-c", "echo 'Triggered at:' $(date)"]
          restartPolicy: Never
```

### `resources` (required)

Type: `[]ResourceReference`

List of Kubernetes resources to watch for changes. At least one resource must be specified.

Each ResourceReference contains:

#### `apiVersion` (required)

Type: `string`

The API version of the resource to watch. Examples:

- `v1` - Core API resources (ConfigMap, Secret, Pod, Service, etc.)
- `apps/v1` - Apps API (Deployment, StatefulSet, DaemonSet, etc.)
- `batch/v1` - Batch API (Job, CronJob)
- Custom API groups: `mygroup.example.com/v1`

#### `kind` (required)

Type: `string`

The kind of resource to watch. Must be a valid Kubernetes resource kind. Examples:

- `ConfigMap`
- `Secret`
- `Deployment`
- `StatefulSet`
- `Service`
- `Node`
- Custom resource kinds

#### `name` (required)

Type: `string`

The name of the specific resource to watch.

#### `namespace` (optional)

Type: `string`

The namespace of the resource. Required for namespaced resources, must be omitted for cluster-scoped resources.

**Validation**:

- For namespaced resources (ConfigMap, Secret, Deployment, etc.): namespace is required
- For cluster-scoped resources (Node, ClusterRole, etc.): namespace must not be specified
- The webhook will validate this during creation/update

#### `fields` (optional)

Type: `[]string`

List of field paths to watch. If not specified or set to `["*"]`, the entire resource is watched.

**Field Path Syntax**:

- Use JSONPath notation to specify fields
- Examples:
  - `"*"` - Watch entire resource (default)
  - `"data"` - Watch specific top-level field
  - `"spec.replicas"` - Watch nested field
  - `"spec.template.spec.containers[*].image"` - Watch all container images
  - `"metadata.labels"` - Watch labels

**Example**:

```yaml
spec:
  resources:
    # Watch entire ConfigMap
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: default

    # Watch only Secret data field
    - apiVersion: v1
      kind: Secret
      name: app-secret
      namespace: default
      fields:
        - "data"

    # Watch only Deployment replicas and image
    - apiVersion: apps/v1
      kind: Deployment
      name: app
      namespace: default
      fields:
        - "spec.replicas"
        - "spec.template.spec.containers[*].image"

    # Watch cluster-scoped Node (no namespace)
    - apiVersion: v1
      kind: Node
      name: worker-1
```

### `condition` (optional)

Type: `string`  
Default: `"Any"`  
Enum: `"Any"`, `"All"`

Determines when to trigger the job based on resource changes:

- **`"Any"`**: Trigger when at least one watched resource changes (OR logic)
- **`"All"`**: Trigger only when all watched resources have changed (AND logic)

**Behavior**:

With `condition: Any`:

```yaml
# If ConfigMap OR Secret changes → trigger
resources:
  - apiVersion: v1
    kind: ConfigMap
    name: config
  - apiVersion: v1
    kind: Secret
    name: secret
condition: Any
```

With `condition: All`:

```yaml
# Only if ConfigMap AND Secret both change → trigger
resources:
  - apiVersion: v1
    kind: ConfigMap
    name: config
  - apiVersion: v1
    kind: Secret
    name: secret
condition: All
```

**Notes**:

- Changes are tracked since the last trigger
- After a trigger, resource hashes are reset
- Useful for coordinating updates across multiple resources

### `cooldown` (optional)

Type: `metav1.Duration`  
Default: `60s`  
Format: Duration string (e.g., `30s`, `5m`, `1h`)

Minimum time to wait between job triggers. Prevents excessive job creation when resources change frequently.

**Valid Formats**:

- Seconds: `30s`, `60s`
- Minutes: `5m`, `10m`
- Hours: `1h`, `2h`
- Combined: `1h30m`, `2h15m30s`

**Example**:

```yaml
spec:
  cooldown: 300s # 5 minutes
```

**Behavior**:

- After a job is triggered, no new jobs will be created for the cooldown period
- Resource changes during cooldown are ignored
- Timer resets after each successful trigger
- Set to `0s` to disable cooldown (not recommended)

### `history` (optional)

Type: `int32`  
Default: `5`  
Minimum: `1`

Maximum number of jobs to keep in history. Older jobs are automatically deleted.

**Example**:

```yaml
spec:
  history: 10 # Keep last 10 jobs
```

**Behavior**:

- Jobs are sorted by creation time (newest first)
- When history limit is exceeded, oldest jobs are deleted
- Applies to both successful and failed jobs
- Jobs are identified by the label `changejob.dev/owner=<name>`

## Status Fields

The status subresource is managed by the controller and reflects the current state of the ChangeTriggeredJob.

### `conditions`

Type: `[]metav1.Condition`

Standard Kubernetes conditions indicating the resource status.

**Common Conditions**:

#### `Available`

- **Type**: `Available`
- **Status**: `True|False|Unknown`
- **Reason**: Various
- **Message**: Human-readable description
- Indicates whether the ChangeTriggeredJob is functioning correctly

#### `Progressing`

- **Type**: `Progressing`
- **Status**: `True|False|Unknown`
- **Reason**: Various
- **Message**: Human-readable description
- Indicates whether work is in progress

#### `Degraded`

- **Type**: `Degraded`
- **Status**: `True|False|Unknown`
- **Reason**: Various
- **Message**: Human-readable description
- Indicates resource or configuration issues

**Example**:

```yaml
status:
  conditions:
    - type: Available
      status: "True"
      lastTransitionTime: "2025-01-15T10:30:00Z"
      reason: ReconciliationSucceeded
      message: ChangeTriggeredJob is available
    - type: Progressing
      status: "False"
      lastTransitionTime: "2025-01-15T10:30:00Z"
      reason: NoChangesDetected
      message: No resource changes detected
```

### `resourceHashes`

Type: `[]ResourceReferenceStatus`

Stores the SHA256 hash of each watched resource's current state. Used for change detection.

**Structure**:

```yaml
status:
  resourceHashes:
    - apiVersion: string
      kind: string
      name: string
      namespace: string
      hash: string # SHA256 hash of resource data
```

**Example**:

```yaml
status:
  resourceHashes:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: default
      hash: "a7f8d3e2b1c4..."
    - apiVersion: v1
      kind: Secret
      name: app-secret
      namespace: default
      hash: "b8e9d4f3c2a5..."
```

### `lastTriggeredTime`

Type: `metav1.Time`

Timestamp of the last time a job was triggered.

**Example**:

```yaml
status:
  lastTriggeredTime: "2025-01-15T10:30:00Z"
```

### `lastJobName`

Type: `string`

Name of the most recently created job.

**Example**:

```yaml
status:
  lastJobName: "config-watcher-abc123"
```

### `lastJobStatus`

Type: `string`  
Enum: `Active`, `Succeeded`, `Failed`

Status of the most recently created job.

- **`Active`**: Job is currently running
- **`Succeeded`**: Job completed successfully
- **`Failed`**: Job failed

**Example**:

```yaml
status:
  lastJobStatus: "Succeeded"
```

## Types Reference

### ResourceReference

```go
type ResourceReference struct {
    // API version of the resource (e.g., "v1", "apps/v1")
    APIVersion string `json:"apiVersion"`

    // Kind of the resource (e.g., "ConfigMap", "Deployment")
    Kind string `json:"kind"`

    // Name of the resource
    Name string `json:"name"`

    // Namespace of the resource (required for namespaced resources)
    // +optional
    Namespace string `json:"namespace,omitempty"`

    // Fields to watch (JSONPath format)
    // If empty or ["*"], watches entire resource
    // +optional
    Fields []string `json:"fields,omitempty"`
}
```

### ResourceReferenceStatus

```go
type ResourceReferenceStatus struct {
    // API version of the resource
    APIVersion string `json:"apiVersion"`

    // Kind of the resource
    Kind string `json:"kind"`

    // Name of the resource
    Name string `json:"name"`

    // Namespace of the resource
    // +optional
    Namespace string `json:"namespace,omitempty"`

    // SHA256 hash of the resource state
    Hash string `json:"hash"`
}
```

### ChangeTriggeredJobSpec

```go
type ChangeTriggeredJobSpec struct {
    // JobTemplate defines the Job to create when triggered
    JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate"`

    // Resources is a list of resources to watch for changes
    Resources []ResourceReference `json:"resources"`

    // Condition determines when to trigger: "Any" or "All"
    // +optional
    // +kubebuilder:default="Any"
    Condition *TriggerCondition `json:"condition,omitempty"`

    // Cooldown is the minimum time between triggers
    // +optional
    // +kubebuilder:default="60s"
    Cooldown *metav1.Duration `json:"cooldown,omitempty"`

    // History is the number of jobs to keep
    // +optional
    // +kubebuilder:default=5
    // +kubebuilder:validation:Minimum=1
    History *int32 `json:"history,omitempty"`
}
```

### ChangeTriggeredJobStatus

```go
type ChangeTriggeredJobStatus struct {
    // Conditions represent the latest available observations
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // ResourceHashes stores the hash of each watched resource
    ResourceHashes []ResourceReferenceStatus `json:"resourceHashes,omitempty"`

    // LastTriggeredTime is when the last job was triggered
    // +optional
    LastTriggeredTime *metav1.Time `json:"lastTriggeredTime,omitempty"`

    // LastJobName is the name of the last created job
    // +optional
    LastJobName string `json:"lastJobName,omitempty"`

    // LastJobStatus is the status of the last job
    // +optional
    LastJobStatus JobState `json:"lastJobStatus,omitempty"`
}
```

## Examples

### Basic ConfigMap Watcher

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: config-watcher
  namespace: default
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: notify
              image: busybox:latest
              command: ["echo", "Config changed!"]
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: default
```

### Multiple Resources with "All" Condition

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: multi-resource-sync
  namespace: production
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: sync
              image: my-sync-tool:v1.0
              command: ["sync", "--all"]
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: production
    - apiVersion: v1
      kind: Secret
      name: app-secret
      namespace: production
    - apiVersion: apps/v1
      kind: Deployment
      name: app
      namespace: production
  condition: All
  cooldown: 300s
  history: 10
```

### Watching Specific Fields

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: image-update-watcher
  namespace: default
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: notify
              image: notification-service:latest
              env:
                - name: MESSAGE
                  value: "Container image updated"
          restartPolicy: Never
  resources:
    - apiVersion: apps/v1
      kind: Deployment
      name: web-app
      namespace: default
      fields:
        - "spec.template.spec.containers[*].image"
  cooldown: 60s
```

### Cluster-Scoped Resource Watcher

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
          serviceAccountName: node-watcher-sa
          containers:
            - name: alert
              image: alert-tool:latest
              command: ["send-alert", "Node configuration changed"]
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: Node
      name: worker-node-1
      # No namespace - Node is cluster-scoped
  cooldown: 300s
```

## Validation Rules

The following validation rules are enforced by webhooks:

1. **Resources List**: Must contain at least one resource
2. **Resource Kind**: Must be a valid Kubernetes resource kind
3. **Resource Namespace**:
   - Required for namespaced resources
   - Must not be set for cluster-scoped resources
4. **Condition**: Must be "Any" or "All"
5. **History**: Must be >= 1
6. **Job Template**: Must contain valid Job specification

## Annotations

The following annotation is automatically added by the mutating webhook:

- `changetriggeredjobs.triggers.changejob.dev/changed-at`: Timestamp of last modification

## Labels

Jobs created by ChangeTriggeredJob automatically receive the following label:

- `changejob.dev/owner`: Set to the name of the ChangeTriggeredJob

This label is used for:

- Identifying owned jobs
- History management and cleanup
- Monitoring and filtering

## RBAC Requirements

To use ChangeTriggeredJob, you need the following permissions:

### For the Controller

```yaml
# ChangeTriggeredJob resources
- apiGroups: ["triggers.changejob.dev"]
  resources:
    [
      "changetriggeredjobs",
      "changetriggeredjobs/status",
      "changetriggeredjobs/finalizers",
    ]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Jobs
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Watched resources (adjust based on what you're watching)
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
```

### For Users

```yaml
# To create and manage ChangeTriggeredJobs
- apiGroups: ["triggers.changejob.dev"]
  resources: ["changetriggeredjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# To view status
- apiGroups: ["triggers.changejob.dev"]
  resources: ["changetriggeredjobs/status"]
  verbs: ["get"]
```

## See Also

- [User Guide](user-guide) - Complete usage guide with examples
- [Installation Guide](installation) - Installation instructions
- [Configuration](configuration) - Controller configuration
- [Examples](examples) - Real-world usage examples
