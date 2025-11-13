# ByteFreezer On-Premises Kubernetes Deployment

This directory contains Kubernetes manifests for deploying ByteFreezer components in an on-premises, self-hosted environment.

## Architecture

In the on-prem deployment model:
- **Customer hosts**: proxy, receiver, piper, packer, PostgreSQL, MinIO/S3
- **ByteFreezer hosts**: control plane only (SaaS)
- **Data flow**: All data remains on customer premises
- **Management**: Control plane manages configuration and monitoring

## Prerequisites

1. Kubernetes cluster (v1.24+)
2. kubectl configured and connected to your cluster
3. Storage provisioner for PersistentVolumes (for PostgreSQL and MinIO)
4. LoadBalancer support (MetalLB, cloud provider LB, or NodePort)
5. Control plane API endpoint and API key

## Components

- **bytefreezer-proxy**: UDP data collector (port 5140)
- **bytefreezer-receiver**: HTTP webhook receiver (port 8080)
- **bytefreezer-piper**: Data processing pipeline (port 8090)
- **bytefreezer-packer**: Data compression (port 8091)

## Quick Start

### 1. Update Configuration

Edit `base/secrets.yaml`:
- Replace S3 credentials with your MinIO/S3 access keys
- Replace PostgreSQL credentials
- Replace control plane API key

Edit `base/configmap.yaml`:
- Update `CONTROL_SERVICE_URL` with your control plane endpoint
- Update S3 endpoint if using external MinIO
- Update PostgreSQL host if using external database

### 2. Deploy Infrastructure (if needed)

If you need PostgreSQL and MinIO in-cluster:
```bash
# Deploy PostgreSQL
kubectl apply -f infrastructure/postgres/

# Deploy MinIO
kubectl apply -f infrastructure/minio/
```

### 3. Deploy ByteFreezer Components

```bash
# Create namespace and base resources
kubectl apply -f base/namespace.yaml
kubectl apply -f base/secrets.yaml
kubectl apply -f base/configmap.yaml

# Deploy all components
kubectl apply -f proxy/
kubectl apply -f receiver/
kubectl apply -f piper/
kubectl apply -f packer/
```

### 4. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n bytefreezer

# Check services
kubectl get svc -n bytefreezer

# Check logs
kubectl logs -n bytefreezer -l app.kubernetes.io/name=bytefreezer-receiver --tail=50
```

## Scaling

Scale individual components based on workload:

```bash
# Scale piper workers
kubectl scale deployment bytefreezer-piper -n bytefreezer --replicas=5

# Scale receiver for high ingestion
kubectl scale deployment bytefreezer-receiver -n bytefreezer --replicas=4

# Scale proxy for high UDP traffic
kubectl scale deployment bytefreezer-proxy -n bytefreezer --replicas=10
```

## Resource Requirements

Minimum recommended resources per replica:

| Component | CPU Request | CPU Limit | Memory Request | Memory Limit |
|-----------|-------------|-----------|----------------|--------------|
| Proxy     | 250m        | 500m      | 256Mi          | 512Mi        |
| Receiver  | 250m        | 500m      | 256Mi          | 512Mi        |
| Piper     | 500m        | 2000m     | 512Mi          | 2Gi          |
| Packer    | 500m        | 1000m     | 512Mi          | 1Gi          |

## Monitoring

Each component exposes Prometheus metrics on port 9090:

```bash
# Port-forward to access metrics
kubectl port-forward -n bytefreezer svc/bytefreezer-piper 9090:9090
```

Access metrics at: http://localhost:9090/metrics

## Troubleshooting

### Pods not starting
```bash
kubectl describe pod -n bytefreezer <pod-name>
kubectl logs -n bytefreezer <pod-name>
```

### Connection issues to control plane
```bash
# Test from within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -n bytefreezer -- \
  curl -H "X-API-Key: YOUR_API_KEY" https://control.bytefreezer.com/health
```

### S3/MinIO connectivity
```bash
# Verify MinIO is accessible
kubectl run -it --rm debug --image=minio/mc --restart=Never -n bytefreezer -- \
  mc alias set myminio http://minio.bytefreezer.svc.cluster.local:9000 ACCESS_KEY SECRET_KEY
```

## Upgrades

To upgrade components:

```bash
# Update image version in deployment files, then:
kubectl apply -f <component>/deployment.yaml

# Or use rolling update
kubectl set image deployment/bytefreezer-piper -n bytefreezer \
  piper=bytefreezer/bytefreezer-piper:v1.1.0
```

## Uninstall

```bash
# Remove all ByteFreezer components
kubectl delete namespace bytefreezer
```

## Support

For issues with:
- On-prem deployment: Contact ByteFreezer support
- Kubernetes cluster: Contact your infrastructure team
- Network connectivity: Check firewall rules and network policies
