---
layout: default
title: Configuration
nav_order: 5
---

# Configuration Guide

This guide covers all configuration options for the kube-changejob controller.

## Table of Contents

- [Controller Configuration](#controller-configuration)
- [Deployment Configuration](#deployment-configuration)
- [RBAC Configuration](#rbac-configuration)
- [Webhook Configuration](#webhook-configuration)
- [Monitoring Configuration](#monitoring-configuration)
- [Security Configuration](#security-configuration)

## Controller Configuration

The controller supports configuration through command-line flags and environment variables.

### Polling Configuration

#### Poll Interval

Controls how frequently the controller checks watched resources for changes.

**Command-line flag**: `--poll-interval`  
**Environment variable**: `POLL_INTERVAL`  
**Default**: `60s`  
**Format**: Duration string (e.g., `30s`, `5m`, `1h`)

**Configuration methods**:

```bash
# Method 1: Environment variable
kubectl set env deployment/kube-changejob-controller-manager \
  -n kube-changejob-system \
  POLL_INTERVAL=30s

# Method 2: Command-line flag
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--poll-interval=30s"
  }]'
```

**Recommendations**:

- **Development**: 30s - Fast feedback
- **Production**: 60s-120s - Balanced performance
- **Large clusters**: 120s-300s - Reduce API server load
- **Low-priority triggers**: 300s+ - Minimize overhead

### Logging Configuration

#### Log Level

Controls the verbosity of controller logs.

**Command-line flag**: `--log-level`  
**Default**: `info`  
**Options**: `debug`, `info`, `warn`, `error`, `panic`, `fatal`

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--log-level=debug"
  }]'
```

**Level descriptions**:

- `debug`: Detailed debugging information
- `info`: General informational messages (default)
- `warn`: Warning messages
- `error`: Error messages only
- `panic`: Panic level errors
- `fatal`: Fatal errors only

#### Log Format

Controls the output format of logs.

**Command-line flag**: `--log-format`  
**Default**: `text`  
**Options**: `text`, `json`

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--log-format=json"
  }]'
```

**When to use**:

- `text`: Human-readable, good for development
- `json`: Structured logs, ideal for log aggregation systems (Elasticsearch, Splunk, etc.)

#### Log Timestamp Format

Controls the timestamp format in logs.

**Command-line flag**: `--log-timestamp`  
**Default**: `iso8601`  
**Options**: `epoch`, `millis`, `nano`, `iso8601`, `rfc3339`

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--log-timestamp=rfc3339"
  }]'
```

### High Availability Configuration

#### Leader Election

Enables leader election for high availability deployments.

**Command-line flag**: `--leader-elect`  
**Default**: `false`

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--leader-elect"
  }]'
```

**Additional leader election flags**:

```bash
--leader-elect-lease-duration=15s    # Lease duration
--leader-elect-renew-deadline=10s    # Renew deadline
--leader-elect-retry-period=2s       # Retry period
```

**Full HA configuration**:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-changejob-controller-manager
  namespace: kube-changejob-system
spec:
  replicas: 3 # Multiple replicas
  template:
    spec:
      containers:
        - name: manager
          args:
            - --leader-elect
            - --leader-elect-lease-duration=15s
            - --leader-elect-renew-deadline=10s
            - --leader-elect-retry-period=2s
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                topologyKey: kubernetes.io/hostname
                labelSelector:
                  matchLabels:
                    control-plane: controller-manager
```

### Metrics Configuration

#### Metrics Bind Address

Address for the metrics server to listen on.

**Command-line flag**: `--metrics-bind-address`  
**Default**: `0` (disabled)  
**Example**: `:8443` (HTTPS) or `:8080` (HTTP)

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--metrics-bind-address=:8443"
  }]'
```

#### Metrics Certificate Path

Directory containing TLS certificates for metrics endpoint.

**Command-line flag**: `--metrics-cert-path`  
**Default**: None (uses HTTP if not specified)

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--metrics-cert-path=/tmp/k8s-metrics-server/metrics-certs"
  }]'
```

### Health Probe Configuration

#### Health Probe Bind Address

Address for health and readiness probes.

**Command-line flag**: `--health-probe-bind-address`  
**Default**: `:8081`

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--health-probe-bind-address=:9090"
  }]'
```

### Webhook Configuration

#### Webhook Certificate Path

Directory containing webhook TLS certificates.

**Command-line flag**: `--webhook-cert-path`  
**Default**: `/tmp/k8s-webhook-server/serving-certs`

```bash
kubectl patch deployment kube-changejob-controller-manager \
  -n kube-changejob-system \
  --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--webhook-cert-path=/certs/webhook"
  }]'
```

## Deployment Configuration

### Resource Limits

Configure CPU and memory limits based on cluster size:

```yaml
spec:
  template:
    spec:
      containers:
        - name: manager
          resources:
            requests:
              cpu: 100m # Minimum CPU
              memory: 64Mi # Minimum memory
            limits:
              cpu: 1000m # Maximum CPU
              memory: 256Mi # Maximum memory
```

**Recommendations by cluster size**:

| Cluster Size         | CPUs         | Memory        | Notes                      |
| -------------------- | ------------ | ------------- | -------------------------- |
| Small (<50 CTJs)     | 100m / 500m  | 64Mi / 128Mi  | Default settings           |
| Medium (50-200 CTJs) | 200m / 1000m | 128Mi / 256Mi | Recommended for production |
| Large (200+ CTJs)    | 500m / 2000m | 256Mi / 512Mi | High-volume deployments    |

### Replicas and Scaling

```yaml
spec:
  replicas: 3 # Number of controller replicas
```

**Recommendations**:

- **Development**: 1 replica
- **Production**: 3 replicas (with leader election)
- **High Availability**: 3-5 replicas across availability zones

### Pod Affinity

Spread controller pods across nodes:

```yaml
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                topologyKey: kubernetes.io/hostname
                labelSelector:
                  matchLabels:
                    control-plane: controller-manager
```

### Node Selector

Run controller on specific nodes:

```yaml
spec:
  template:
    spec:
      nodeSelector:
        node-role.kubernetes.io/control-plane: ""
```

### Tolerations

Allow controller to run on tainted nodes:

```yaml
spec:
  template:
    spec:
      tolerations:
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
          effect: NoSchedule
```

## RBAC Configuration

### Default RBAC

The controller requires the following permissions:

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

# Watched resources
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
```

### Restricting Watched Resources

For security, limit what resources the controller can watch:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kube-changejob-manager-role-restricted
rules:
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

  # Only allow watching specific resources
  - apiGroups: [""]
    resources: ["configmaps", "secrets", "services"]
    verbs: ["get", "list", "watch"]

  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets"]
    verbs: ["get", "list", "watch"]
```

### User RBAC

Create roles for users to manage ChangeTriggeredJobs:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: changetriggeredjob-editor
  namespace: default
rules:
  - apiGroups: ["triggers.changejob.dev"]
    resources: ["changetriggeredjobs"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: changetriggeredjob-editor-binding
  namespace: default
subjects:
  - kind: User
    name: jane@example.com
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: changetriggeredjob-editor
  apiGroup: rbac.authorization.k8s.io
```

## Webhook Configuration

### TLS Certificates

Webhooks require TLS certificates. cert-manager handles this automatically:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: kube-changejob-serving-cert
  namespace: kube-changejob-system
spec:
  dnsNames:
    - kube-changejob-webhook-service.kube-changejob-system.svc
    - kube-changejob-webhook-service.kube-changejob-system.svc.cluster.local
  issuerRef:
    kind: Issuer
    name: kube-changejob-selfsigned-issuer
  secretName: webhook-server-cert
```

### Webhook Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kube-changejob-webhook-service
  namespace: kube-changejob-system
spec:
  ports:
    - port: 443
      targetPort: 9443
      protocol: TCP
  selector:
    control-plane: controller-manager
```

### Disabling Webhooks

For development or troubleshooting:

```bash
export ENABLE_WEBHOOKS=false
make run
```

Or remove webhook configurations:

```bash
kubectl delete validatingwebhookconfiguration kube-changejob-validating-webhook-configuration
kubectl delete mutatingwebhookconfiguration kube-changejob-mutating-webhook-configuration
```

## Monitoring Configuration

### Prometheus Metrics

Enable Prometheus monitoring:

```bash
kubectl apply -k config/prometheus
```

This creates a ServiceMonitor resource:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kube-changejob-controller-manager-metrics-monitor
  namespace: kube-changejob-system
spec:
  endpoints:
    - path: /metrics
      port: https
      scheme: https
      tlsConfig:
        insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
```

### Available Metrics

Standard controller-runtime metrics:

- `controller_runtime_reconcile_total`: Total reconciliation attempts
- `controller_runtime_reconcile_errors_total`: Total reconciliation errors
- `controller_runtime_reconcile_time_seconds`: Reconciliation latency
- `workqueue_depth`: Work queue depth
- `workqueue_adds_total`: Work queue additions

### Custom Dashboards

Create Grafana dashboards using these metrics:

```promql
# Reconciliation rate
rate(controller_runtime_reconcile_total{controller="changetriggeredjob"}[5m])

# Error rate
rate(controller_runtime_reconcile_errors_total{controller="changetriggeredjob"}[5m])

# Reconciliation latency (95th percentile)
histogram_quantile(0.95, rate(controller_runtime_reconcile_time_seconds_bucket[5m]))

# Active ChangeTriggeredJobs
count(kube_changetriggeredjob_info)
```

## Security Configuration

### Pod Security Context

The controller runs with a restrictive security context:

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: manager
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
```

### Network Policies

Enable network policies for additional security:

```bash
kubectl apply -k config/network-policy
```

This restricts traffic to:

- Webhook port (9443)
- Metrics port (8443)
- Health probe port (8081)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kube-changejob-controller-manager
  namespace: kube-changejob-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: kube-system
      ports:
        - protocol: TCP
          port: 9443 # Webhook
    - from:
        - namespaceSelector:
            matchLabels:
              metrics: enabled
      ports:
        - protocol: TCP
          port: 8443 # Metrics
  egress:
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 6443 # Kubernetes API
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: UDP
          port: 53 # DNS
```

### Service Account

The controller uses a dedicated service account:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-changejob-controller-manager
  namespace: kube-changejob-system
automountServiceAccountToken: true
```

## Configuration Examples

### Production Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-changejob-controller-manager
  namespace: kube-changejob-system
spec:
  replicas: 3
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      serviceAccountName: kube-changejob-controller-manager
      containers:
        - name: manager
          image: controller:latest
          args:
            - --leader-elect
            - --poll-interval=60s
            - --log-level=info
            - --log-format=json
            - --metrics-bind-address=:8443
          env:
            - name: POLL_INTERVAL
              value: "60s"
          resources:
            requests:
              cpu: 200m
              memory: 128Mi
            limits:
              cpu: 1000m
              memory: 256Mi
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8081
            initialDelaySeconds: 15
            periodSeconds: 20
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8081
            initialDelaySeconds: 5
            periodSeconds: 10
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                topologyKey: kubernetes.io/hostname
                labelSelector:
                  matchLabels:
                    control-plane: controller-manager
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
```

### Development Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-changejob-controller-manager
  namespace: kube-changejob-system
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: manager
          args:
            - --poll-interval=30s
            - --log-level=debug
            - --log-format=text
          resources:
            requests:
              cpu: 50m
              memory: 32Mi
            limits:
              cpu: 500m
              memory: 128Mi
```

## See Also

- [Installation Guide](installation.md) - Installation instructions
- [User Guide](user-guide.md) - Usage guide
- [API Reference](api-reference.md) - API documentation
- [Examples](examples.md) - Example configurations
