# ByteFreezer Control + LocalStack Integration

Complete integration guide for connecting ByteFreezer Control to LocalStack in your K3s cluster.

## 🎯 **Overview**

This integration connects your ByteFreezer Control service to the LocalStack AWS emulator, providing:
- **Centralized AWS Services** - All ByteFreezer services share the same LocalStack instance
- **Service Discovery** - Automatic LocalStack endpoint discovery via Kubernetes DNS
- **Environment Variables** - Pre-configured AWS credentials and endpoints
- **Health Monitoring** - Integrated health checks for both services
- **Development Workflow** - Streamlined dev/staging deployment patterns

## 🚀 **Quick Integration**

### **Prerequisites**
```bash
# 1. Ensure LocalStack is deployed in your K3s cluster
cd ../bytefreezer-localstack
./deploy.sh

# 2. Verify LocalStack is running
kubectl get pods -n localstack
```

### **Deploy ByteFreezer Control with LocalStack**

**Option 1: Ansible (AWX Compatible) - for server/VM deployment**
```bash
cd ansible
ansible-playbook playbooks/install.yml \
  -e target_environment=development \
  -e enable_debug_logging=true
```

**Option 2: Helm (Manual K3s) - for Kubernetes deployment**
```bash
helm upgrade --install bytefreezer-control ./helm/bytefreezer-control \
  --set environment=development \
  --set localstack.enabled=true \
  --set persistence.enabled=false
```

### **Test Integration**
```bash
# Run comprehensive integration test
./test-localstack-integration.sh
```

## 🔧 **Configuration Details**

### **Service Discovery**
ByteFreezer Control automatically discovers LocalStack using Kubernetes DNS:

```yaml
# Automatic service discovery endpoints
localstack:
  endpoint: "http://localstack-internal.localstack.svc.cluster.local:4566"
  admin_endpoint: "http://localstack-internal.localstack.svc.cluster.local:4510"
  region: "us-east-1"
  access_key_id: "test"
  secret_access_key: "test"
```

### **Environment Variables**
The following AWS environment variables are automatically configured:

```bash
AWS_ENDPOINT_URL=http://localstack-internal.localstack.svc.cluster.local:4566
AWS_DEFAULT_REGION=us-east-1
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
LOCALSTACK_ENDPOINT=http://localstack-internal.localstack.svc.cluster.local:4566
```

### **Network Policies**
Network policies allow communication between services:

```yaml
# ByteFreezer Control → LocalStack
egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: localstack
    ports:
    - protocol: TCP
      port: 4566  # LocalStack edge port
    - protocol: TCP
      port: 4510  # LocalStack admin port
```

## 🎭 **Deployment Options**

### **Ansible (AWX Compatible)**
For traditional server/VM deployments:
- **Standard playbooks** that can be imported as AWX Job Templates
- **Group variables** for environment-specific configuration
- **Survey variables** support through `--extra-vars`
- **Inventory-based** deployment targeting

**AWX Job Template Variables:**
- **target_environment**: `development` | `staging` | `production`
- **enable_debug_logging**: Debug mode for troubleshooting

### **Helm (Manual K3s)**
For Kubernetes deployments:
- **Production-ready** Helm chart with all necessary resources
- **LocalStack integration** with automatic service discovery
- **Persistent storage** options for data retention
- **Network policies** for secure communication

**Key Features:**
- Configurable via values.yaml or --set flags
- Built-in health checks and readiness probes
- Resource limits and autoscaling support
- Separate from AWX workflow for manual control

## 🧪 **Testing and Validation**

### **Integration Test Script**
Run comprehensive integration tests:

```bash
./test-localstack-integration.sh
```

**Test Coverage:**
- ✅ Kubernetes cluster connectivity
- ✅ LocalStack deployment and health
- ✅ Service discovery (DNS resolution)
- ✅ Network connectivity between services
- ✅ AWS service availability (S3, SQS, etc.)
- ✅ ByteFreezer Control deployment
- ✅ Environment variable configuration
- ✅ Health endpoint accessibility

### **Manual Testing**
```bash
# 1. Port forward services
kubectl port-forward svc/localstack-external 4566:4566 -n localstack &
kubectl port-forward svc/bytefreezer-control 8082:8082 -n bytefreezer-control &

# 2. Test LocalStack
curl http://localhost:4566/_localstack/health

# 3. Test ByteFreezer Control
curl http://localhost:8082/api/v1/health
curl http://localhost:8082/api/v1/config

# 4. Test AWS services via LocalStack
aws --endpoint-url=http://localhost:4566 s3 ls
aws --endpoint-url=http://localhost:4566 sqs list-queues
```

### **AWS CLI Testing**
```bash
# Configure AWS CLI for LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_DEFAULT_REGION=us-east-1
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Test AWS services
aws s3 mb s3://test-bucket
aws s3 ls
aws sqs create-queue --queue-name test-queue
aws sqs list-queues
```

## 🔍 **Troubleshooting**

### **Common Issues**

**LocalStack Not Accessible:**
```bash
# Check LocalStack pods
kubectl get pods -n localstack

# Check LocalStack logs
kubectl logs -l app=localstack -n localstack

# Test connectivity
kubectl run test-pod --image=curlimages/curl --rm -i -- \
  curl http://localstack-internal.localstack.svc.cluster.local:4566/_localstack/health
```

**ByteFreezer Control Can't Reach LocalStack:**
```bash
# Check network policies
kubectl get networkpolicy -n bytefreezer-control

# Check DNS resolution
kubectl exec -n bytefreezer-control deployment/bytefreezer-control -- \
  nslookup localstack-internal.localstack.svc.cluster.local

# Check environment variables
kubectl exec -n bytefreezer-control deployment/bytefreezer-control -- env | grep AWS
```

**Service Discovery Issues:**
```bash
# Check service endpoints
kubectl get svc -n localstack
kubectl get endpoints -n localstack

# Test cross-namespace connectivity
kubectl run debug --image=busybox -n bytefreezer-control --rm -i -- \
  wget -qO- http://localstack-internal.localstack.svc.cluster.local:4566/_localstack/health
```

### **Debug Mode**
Enable debug logging for detailed troubleshooting:

```bash
# Deploy with debug logging
ansible-playbook playbooks/kubernetes/deploy.yml \
  -e enable_debug_logging=true \
  -e target_environment=development

# Check debug logs
kubectl logs -f deployment/bytefreezer-control -n bytefreezer-control
```

## 📊 **Monitoring Integration**

### **Health Endpoints**
- **LocalStack**: `http://localstack-internal.localstack.svc.cluster.local:4566/_localstack/health`
- **Control**: `http://bytefreezer-control.bytefreezer-control.svc.cluster.local:8082/api/v1/health`

### **Service Mesh Integration**
If using Istio service mesh:

```bash
# Check service mesh traffic
kubectl port-forward svc/kiali 20001:20001 -n istio-system
# Visit: http://localhost:20001

# View distributed tracing
kubectl port-forward svc/jaeger 16686:16686 -n istio-system
# Visit: http://localhost:16686
```

### **Metrics Collection**
Prometheus metrics endpoints:
- **LocalStack**: `http://localstack-internal.localstack.svc.cluster.local:4566/_localstack/metrics`
- **Control**: `http://bytefreezer-control.bytefreezer-control.svc.cluster.local:8082/metrics`

## 🔄 **Development Workflow**

### **Local Development**
```bash
# 1. Start port forwards
kubectl port-forward svc/localstack-external 4566:4566 -n localstack &

# 2. Configure local environment
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_DEFAULT_REGION=us-east-1
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# 3. Develop ByteFreezer Control locally
cd /path/to/bytefreezer-control
go run . --config config.yaml
```

### **Hot Reload with DevSpace**
```bash
# Use DevSpace for live development
cd ../bytefreezer-localstack/devspace
devspace dev --var LOCALSTACK_ENDPOINT=localstack-internal.localstack.svc.cluster.local:4566
```

### **CI/CD Integration**
```bash
# Example CI/CD pipeline step
- name: Test Integration
  run: |
    ./test-localstack-integration.sh
    kubectl wait --for=condition=available deployment/bytefreezer-control -n bytefreezer-control
    curl -f http://localhost:8082/api/v1/health
```

## 🎯 **Production Considerations**

### **High Availability**
- Deploy multiple LocalStack replicas for redundancy
- Use persistent storage for LocalStack data
- Configure proper resource limits and requests

### **Security**
- Use proper AWS credentials (not test/test)
- Implement network policies for production
- Enable TLS between services

### **Backup and Recovery**
```bash
# Backup LocalStack data
kubectl exec -n localstack deployment/localstack -- \
  tar czf /tmp/localstack-backup.tar.gz /tmp/localstack/data

# Restore LocalStack data
kubectl cp localstack-backup.tar.gz localstack/pod-name:/tmp/
```

This integration provides a robust foundation for connecting ByteFreezer Control to LocalStack with comprehensive testing, monitoring, and deployment automation! 🚀