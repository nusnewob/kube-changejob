---
layout: default
title: Examples
nav_order: 6
description: "Real-world usage examples and patterns"
---

# Examples

This document provides real-world examples of using kube-changejob for various use cases.

## Table of Contents

- [Basic Examples](#basic-examples)
- [Configuration Management](#configuration-management)
- [Deployment Automation](#deployment-automation)
- [Monitoring and Alerting](#monitoring-and-alerting)
- [Backup and Recovery](#backup-and-recovery)
- [Security and Compliance](#security-and-compliance)
- [Advanced Patterns](#advanced-patterns)

## Basic Examples

### Simple ConfigMap Watcher

Trigger a job when a ConfigMap changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: configmap-watcher
  namespace: default
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: logger
              image: busybox:latest
              command:
                - sh
                - -c
                - |
                  echo "ConfigMap changed at: $(date)"
                  echo "Triggering application reload..."
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: default
  cooldown: 60s
  history: 5
```

### Secret Watcher

Monitor a Secret and trigger when it changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: secret-watcher
  namespace: default
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: notifier
              image: busybox:latest
              command:
                - sh
                - -c
                - |
                  echo "Secret updated at: $(date)"
                  echo "Please rotate credentials in dependent services"
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: Secret
      name: database-credentials
      namespace: default
  cooldown: 300s
```

## Configuration Management

### Multi-Environment Config Sync

Sync configurations across environments when changes are detected:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: config-sync-production
  namespace: production
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: config-sync-sa
          containers:
            - name: sync
              image: my-registry/config-sync:v1.0
              command:
                - config-sync
              args:
                - --source-namespace=production
                - --target-systems=app1,app2,app3
                - --verify
              env:
                - name: SYNC_MODE
                  value: "incremental"
                - name: DRY_RUN
                  value: "false"
              volumeMounts:
                - name: config
                  mountPath: /config
                  readOnly: true
          volumes:
            - name: config
              configMap:
                name: app-config
          restartPolicy: OnFailure
      backoffLimit: 3
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: production
      fields:
        - "data"
    - apiVersion: v1
      kind: Secret
      name: app-secrets
      namespace: production
      fields:
        - "data"
  condition: All # Sync only when both config and secrets are updated
  cooldown: 600s
  history: 10
```

### Application Reload on Config Change

Reload application pods when configuration changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: app-reload-trigger
  namespace: default
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: pod-restarter
          containers:
            - name: reload
              image: bitnami/kubectl:latest
              command:
                - sh
                - -c
                - |
                  echo "Rolling restart of deployment: web-app"
                  kubectl rollout restart deployment/web-app -n default
                  kubectl rollout status deployment/web-app -n default
          restartPolicy: OnFailure
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: web-app-config
      namespace: default
  cooldown: 120s
```

## Deployment Automation

### Image Update Notifications

Send notifications when container images are updated:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: image-update-notifier
  namespace: production
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: notify
              image: my-registry/slack-notifier:v1.0
              env:
                - name: SLACK_WEBHOOK_URL
                  valueFrom:
                    secretKeyRef:
                      name: slack-webhook
                      key: url
                - name: DEPLOYMENT_NAME
                  value: "web-app"
                - name: NAMESPACE
                  value: "production"
              command:
                - notify-slack
              args:
                - --message
                - "🚀 Deployment web-app image updated in production"
                - --channel
                - "#deployments"
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

### Post-Deployment Validation

Run validation tests after deployment changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: deployment-validator
  namespace: production
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: validator-sa
          containers:
            - name: validate
              image: my-registry/validator:v1.0
              command:
                - validate-deployment
              args:
                - --deployment=web-app
                - --namespace=production
                - --tests=smoke,integration
                - --timeout=5m
              env:
                - name: VALIDATION_MODE
                  value: "strict"
                - name: ALERT_ON_FAILURE
                  value: "true"
          restartPolicy: Never
      backoffLimit: 2
  resources:
    - apiVersion: apps/v1
      kind: Deployment
      name: web-app
      namespace: production
      fields:
        - "spec.replicas"
        - "spec.template.spec.containers[*].image"
  cooldown: 300s
  history: 15
```

### Canary Deployment Automation

Trigger canary analysis when deployment changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: canary-analyzer
  namespace: production
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: canary-sa
          containers:
            - name: analyze
              image: my-registry/canary-analyzer:v1.0
              command:
                - analyze-canary
              args:
                - --deployment=web-app
                - --canary-weight=10
                - --duration=10m
                - --metrics=error-rate,latency,throughput
                - --success-threshold=99
              env:
                - name: PROMETHEUS_URL
                  value: "http://prometheus.monitoring:9090"
                - name: AUTO_PROMOTE
                  value: "false"
          restartPolicy: Never
  resources:
    - apiVersion: apps/v1
      kind: Deployment
      name: web-app-canary
      namespace: production
  cooldown: 900s # 15 minutes between canary tests
```

## Monitoring and Alerting

### Prometheus Rule Updates

Reload Prometheus when alert rules change:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: prometheus-reload
  namespace: monitoring
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: reload
              image: curlimages/curl:latest
              command:
                - sh
                - -c
                - |
                  echo "Reloading Prometheus configuration..."
                  curl -X POST http://prometheus-server.monitoring:9090/-/reload
                  echo "Prometheus reloaded successfully"
          restartPolicy: OnFailure
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: prometheus-rules
      namespace: monitoring
      fields:
        - "data"
  cooldown: 60s
```

### Alert Manager Configuration Sync

Update Alert Manager when configuration changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: alertmanager-sync
  namespace: monitoring
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: alertmanager-reloader
          containers:
            - name: reload
              image: bitnami/kubectl:latest
              command:
                - sh
                - -c
                - |
                  echo "Syncing AlertManager configuration..."
                  kubectl delete pod -l app=alertmanager -n monitoring
                  echo "AlertManager pods restarted"
          restartPolicy: OnFailure
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: alertmanager-config
      namespace: monitoring
    - apiVersion: v1
      kind: Secret
      name: alertmanager-secret
      namespace: monitoring
  condition: Any
  cooldown: 120s
```

### Node Change Alerts

Alert when node configuration changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: node-change-alert
  namespace: kube-changejob-system
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: cluster-monitor
          containers:
            - name: alert
              image: my-registry/alert-sender:v1.0
              env:
                - name: ALERT_TYPE
                  value: "node-configuration-change"
                - name: SEVERITY
                  value: "warning"
                - name: PAGERDUTY_KEY
                  valueFrom:
                    secretKeyRef:
                      name: pagerduty-key
                      key: integration-key
              command:
                - send-alert
              args:
                - --message
                - "Node worker-1 configuration has changed"
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: Node
      name: worker-1
      fields:
        - "metadata.labels"
        - "spec.taints"
  cooldown: 300s
```

## Backup and Recovery

### Database Backup Trigger

Trigger database backups when credentials or configuration changes:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: database-backup
  namespace: database
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: backup-sa
          containers:
            - name: backup
              image: my-registry/pg-backup:v1.0
              command:
                - pg-backup
              args:
                - --format=custom
                - --compress=9
                - --jobs=4
              env:
                - name: PGHOST
                  valueFrom:
                    configMapKeyRef:
                      name: database-config
                      key: host
                - name: PGUSER
                  valueFrom:
                    secretKeyRef:
                      name: database-credentials
                      key: username
                - name: PGPASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: database-credentials
                      key: password
                - name: BACKUP_LOCATION
                  value: "s3://backups/postgres/incremental"
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
  cooldown: 3600s # Maximum one backup per hour
  history: 24
```

### Disaster Recovery Testing

Trigger DR tests when critical resources change:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: dr-test-trigger
  namespace: infrastructure
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: dr-tester
          containers:
            - name: test
              image: my-registry/dr-tester:v1.0
              command:
                - dr-test
              args:
                - --mode=validation
                - --skip-destructive
                - --report-to=slack
              env:
                - name: DR_ENVIRONMENT
                  value: "staging"
                - name: TEST_SUITE
                  value: "comprehensive"
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: dr-config
      namespace: infrastructure
  cooldown: 86400s # Once per day maximum
```

## Security and Compliance

### Certificate Rotation

Trigger certificate rotation workflows:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: cert-rotation-trigger
  namespace: cert-manager
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: cert-rotator
          containers:
            - name: rotate
              image: my-registry/cert-rotator:v1.0
              command:
                - rotate-certs
              args:
                - --certificate=web-app-tls
                - --namespace=production
                - --notify-services
              env:
                - name: ROTATION_MODE
                  value: "rolling"
                - name: VERIFICATION_ENABLED
                  value: "true"
          restartPolicy: OnFailure
      backoffLimit: 2
  resources:
    - apiVersion: cert-manager.io/v1
      kind: Certificate
      name: web-app-tls
      namespace: production
      fields:
        - "status.notAfter"
        - "status.renewalTime"
  cooldown: 3600s
```

### Policy Compliance Checks

Run compliance checks when policies change:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: compliance-checker
  namespace: compliance
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: compliance-checker
          containers:
            - name: check
              image: my-registry/policy-checker:v1.0
              command:
                - check-compliance
              args:
                - --policies=/policies
                - --resources=all
                - --output-format=json
                - --fail-on-violation=false
              volumeMounts:
                - name: policies
                  mountPath: /policies
                  readOnly: true
              env:
                - name: COMPLIANCE_FRAMEWORK
                  value: "PCI-DSS,SOC2"
                - name: REPORT_DESTINATION
                  value: "s3://compliance-reports/"
          volumes:
            - name: policies
              configMap:
                name: security-policies
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: security-policies
      namespace: compliance
  cooldown: 7200s
  history: 30
```

### Secrets Scanning

Scan for exposed secrets when resources change:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: secrets-scanner
  namespace: security
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: secrets-scanner
          containers:
            - name: scan
              image: my-registry/secrets-scanner:v1.0
              command:
                - scan-secrets
              args:
                - --namespace=production
                - --scan-depth=deep
                - --patterns=aws,gcp,github,slack
              env:
                - name: ALERT_ON_DETECTION
                  value: "true"
                - name: REMEDIATION_MODE
                  value: "alert-only"
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: app-config
      namespace: production
    - apiVersion: apps/v1
      kind: Deployment
      name: web-app
      namespace: production
      fields:
        - "spec.template.spec.containers[*].env"
  condition: Any
  cooldown: 300s
```

## Advanced Patterns

### Chained Workflows

Trigger multiple dependent workflows:

```yaml
# Primary trigger
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: primary-sync
  namespace: default
spec:
  jobTemplate:
    metadata:
      annotations:
        workflow: "primary"
    spec:
      template:
        spec:
          containers:
            - name: sync
              image: my-registry/sync-tool:v1.0
              command:
                - sync-primary
          restartPolicy: Never
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: primary-config
      namespace: default
  cooldown: 120s
---
# Secondary trigger (watches job from primary)
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: secondary-sync
  namespace: default
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: sync
              image: my-registry/sync-tool:v1.0
              command:
                - sync-secondary
          restartPolicy: Never
  resources:
    - apiVersion: batch/v1
      kind: Job
      name: primary-sync-*
      namespace: default
      fields:
        - "status.succeeded"
  cooldown: 60s
```

### Multi-Cluster Synchronization

Sync configurations across multiple clusters:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: multi-cluster-sync
  namespace: federation
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: cluster-admin
          containers:
            - name: sync
              image: my-registry/multi-cluster-sync:v1.0
              command:
                - sync-clusters
              args:
                - --source-cluster=primary
                - --target-clusters=us-east-1,us-west-2,eu-west-1
                - --resource-types=configmaps,secrets
                - --verify
              env:
                - name: SYNC_MODE
                  value: "push"
                - name: CONFLICT_RESOLUTION
                  value: "source-wins"
              volumeMounts:
                - name: kubeconfigs
                  mountPath: /kubeconfigs
                  readOnly: true
          volumes:
            - name: kubeconfigs
              secret:
                secretName: cluster-kubeconfigs
          restartPolicy: OnFailure
      backoffLimit: 3
  resources:
    - apiVersion: v1
      kind: ConfigMap
      name: shared-config
      namespace: federation
  cooldown: 600s
  history: 20
```

### GitOps Integration

Trigger GitOps synchronization:

```yaml
apiVersion: triggers.changejob.dev/v1alpha
kind: ChangeTriggeredJob
metadata:
  name: argocd-sync-trigger
  namespace: argocd
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: argocd-sync
          containers:
            - name: sync
              image: argoproj/argocd:latest
              command:
                - argocd
              args:
                - app
                - sync
                - my-application
                - --prune
                - --force
              env:
                - name: ARGOCD_SERVER
                  value: "argocd-server.argocd:443"
                - name: ARGOCD_AUTH_TOKEN
                  valueFrom:
                    secretKeyRef:
                      name: argocd-token
                      key: token
          restartPolicy: OnFailure
  resources:
    - apiVersion: argoproj.io/v1alpha1
      kind: Application
      name: my-application
      namespace: argocd
      fields:
        - "spec.source.repoURL"
        - "spec.source.targetRevision"
  cooldown: 180s
```

## Testing Examples

All examples in this document are production-ready but should be tested in a development environment first:

```bash
# Create a test namespace
kubectl create namespace test-changejob

# Apply an example
kubectl apply -f <example>.yaml

# Test by modifying the watched resource
kubectl patch configmap <name> -p '{"data":{"test":"value"}}'

# Watch for created jobs
kubectl get jobs -l changejob.dev/owner=<ctj-name> -w

# View job logs
kubectl logs job/<job-name>

# Clean up
kubectl delete changetriggeredjob <name>
```

## See Also

- [User Guide](user-guide) - Complete usage guide
- [API Reference](api-reference) - API specification
- [Configuration](configuration) - Controller configuration
- [Installation](installation) - Installation guide
- [Release Process](release) - How to create and manage releases
