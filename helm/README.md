# ByteFreezer Control - Helm Chart

Manual Kubernetes/K3s deployment using Helm for ByteFreezer Control service.

## Prerequisites

- Kubernetes cluster (K3s recommended)
- Helm 3.x installed
- kubectl configured for your cluster

## Quick Start

### 1. Install Helm Chart

```bash
# Basic installation
helm install bytefreezer-control ./helm/bytefreezer-control

# With custom values
helm install bytefreezer-control ./helm/bytefreezer-control \
  --set image.tag=v1.0.0 \
  --set environment=production \
  --set config.tenant_id=your-tenant \
  --set config.bearer_token=your-token
```

### 2. LocalStack Integration

For development with LocalStack:

```bash
# Deploy with LocalStack integration enabled
helm install bytefreezer-control ./helm/bytefreezer-control \
  --set localstack.enabled=true \
  --set environment=development \
  --values helm/bytefreezer-control/values-localstack.yaml
```

### 3. Production Deployment

```bash
# Production with persistence and resource limits
helm install bytefreezer-control ./helm/bytefreezer-control \
  --set environment=production \
  --set persistence.enabled=true \
  --set persistence.size=10Gi \
  --set resources.limits.memory=1Gi \
  --set resources.limits.cpu=1000m \
  --set localstack.enabled=false
```

## Configuration

### Basic Configuration

Edit values in `helm/bytefreezer-control/values.yaml`:

```yaml
# Image configuration
image:
  repository: ghcr.io/n0needt0/bytefreezer-control
  tag: "latest"

# Service configuration
service:
  type: ClusterIP
  port: 8082

# Application config
config:
  tenant_id: "your-tenant"
  bearer_token: "your-token"
```

### LocalStack Configuration

For development with LocalStack AWS emulation:

```yaml
localstack:
  enabled: true
  endpoint: "http://localstack-internal.localstack.svc.cluster.local:4566"
  region: "us-east-1"
  access_key_id: "test"
  secret_access_key: "test"
```

### Persistence

Enable data persistence:

```yaml
persistence:
  enabled: true
  storageClass: "local-path"  # K3s default
  size: 5Gi
```

## Upgrade

```bash
# Upgrade to new version
helm upgrade bytefreezer-control ./helm/bytefreezer-control \
  --set image.tag=v1.1.0

# Upgrade with new values
helm upgrade bytefreezer-control ./helm/bytefreezer-control \
  --values my-custom-values.yaml
```

## Uninstall

```bash
# Remove deployment (keeps PVCs)
helm uninstall bytefreezer-control

# Remove everything including data
helm uninstall bytefreezer-control
kubectl delete pvc -l app.kubernetes.io/name=bytefreezer-control
```

## Monitoring

### Health Checks

```bash
# Port forward to access service
kubectl port-forward svc/bytefreezer-control 8082:8082

# Check health
curl http://localhost:8082/api/v2/health
```

### Logs

```bash
# View logs
kubectl logs -l app.kubernetes.io/name=bytefreezer-control -f

# Debug deployment
kubectl describe deployment bytefreezer-control
kubectl get events --sort-by=.metadata.creationTimestamp
```

## Custom Values Files

### Development (values-dev.yaml)
```yaml
environment: development
replicaCount: 1
resources:
  requests:
    memory: 128Mi
    cpu: 100m
localstack:
  enabled: true
persistence:
  enabled: false
```

### Production (values-prod.yaml)
```yaml
environment: production
replicaCount: 2
resources:
  limits:
    memory: 1Gi
    cpu: 1000m
  requests:
    memory: 512Mi
    cpu: 500m
localstack:
  enabled: false
persistence:
  enabled: true
  size: 10Gi
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
```

## Troubleshooting

### Common Issues

**Pod not starting:**
```bash
kubectl describe pod -l app.kubernetes.io/name=bytefreezer-control
kubectl logs -l app.kubernetes.io/name=bytefreezer-control
```

**Service not accessible:**
```bash
kubectl get svc bytefreezer-control
kubectl get endpoints bytefreezer-control
```

**LocalStack connectivity:**
```bash
# Test from pod
kubectl exec -it deployment/bytefreezer-control -- wget -qO- http://localstack-internal.localstack.svc.cluster.local:4566/_localstack/health
```

### Network Policies

If network policies are enabled, ensure:
- LocalStack namespace has label: `name: localstack`
- DNS resolution is allowed
- Required ports are open (4566, 4510)

This Helm chart provides a production-ready deployment option separate from the Ansible AWX automation workflow.