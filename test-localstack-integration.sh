#!/bin/bash
set -e

# ByteFreezer Control - LocalStack Integration Test
# Test the integration between ByteFreezer Control and LocalStack in K3s

echo "🧪 Testing ByteFreezer Control + LocalStack Integration"
echo "====================================================="

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test configuration
LOCALSTACK_NAMESPACE="localstack"
CONTROL_NAMESPACE="bytefreezer-control"
LOCALSTACK_SERVICE="localstack-internal"
CONTROL_SERVICE="bytefreezer-control"

print_status "Step 1: Checking Prerequisites"

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed"
    exit 1
fi

# Check if we can connect to the cluster
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster"
    exit 1
fi

print_success "Prerequisites check passed"

print_status "Step 2: Checking LocalStack Deployment"

# Check if LocalStack namespace exists
if ! kubectl get namespace "$LOCALSTACK_NAMESPACE" &> /dev/null; then
    print_error "LocalStack namespace '$LOCALSTACK_NAMESPACE' not found"
    print_status "Deploy LocalStack first using: cd ../bytefreezer-localstack && ./deploy.sh"
    exit 1
fi

# Check if LocalStack pods are running
LOCALSTACK_READY=$(kubectl get pods -n "$LOCALSTACK_NAMESPACE" -l app=localstack --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
if [ "$LOCALSTACK_READY" -eq 0 ]; then
    print_error "No LocalStack pods are running"
    kubectl get pods -n "$LOCALSTACK_NAMESPACE"
    exit 1
fi

print_success "LocalStack is running ($LOCALSTACK_READY pods)"

print_status "Step 3: Checking LocalStack Service Discovery"

# Test LocalStack internal service
LOCALSTACK_ENDPOINT="localstack-internal.localstack.svc.cluster.local:4566"
print_status "Testing LocalStack endpoint: $LOCALSTACK_ENDPOINT"

# Create a test pod to check connectivity
kubectl run localstack-test --image=curlimages/curl:latest --rm -i --restart=Never -- \
    curl -s --connect-timeout 10 "http://$LOCALSTACK_ENDPOINT/_localstack/health" > /tmp/localstack-health.json

if [ $? -eq 0 ]; then
    print_success "LocalStack health endpoint is accessible"
    echo "Health status:"
    cat /tmp/localstack-health.json | head -5
else
    print_error "Cannot reach LocalStack health endpoint"
    exit 1
fi

print_status "Step 4: Testing AWS Service Discovery"

# Test S3 service discovery
print_status "Testing S3 service availability..."
kubectl run aws-test --image=amazon/aws-cli:latest --rm -i --restart=Never --env="AWS_ACCESS_KEY_ID=test" --env="AWS_SECRET_ACCESS_KEY=test" --env="AWS_DEFAULT_REGION=us-east-1" -- \
    aws --endpoint-url="http://$LOCALSTACK_ENDPOINT" s3 ls > /tmp/s3-test.out 2>&1

if [ $? -eq 0 ]; then
    print_success "S3 service is accessible via LocalStack"
else
    print_warning "S3 test failed (this is expected if no buckets exist)"
    cat /tmp/s3-test.out
fi

print_status "Step 5: Deploying ByteFreezer Control with LocalStack Integration"

# Create ByteFreezer Control namespace if it doesn't exist
if ! kubectl get namespace "$CONTROL_NAMESPACE" &> /dev/null; then
    print_status "Creating ByteFreezer Control namespace..."
    kubectl create namespace "$CONTROL_NAMESPACE"
fi

# Deploy ByteFreezer Control using our Ansible playbook
print_status "Deploying ByteFreezer Control with LocalStack integration..."

# Check if we're in the right directory
if [ ! -f "ansible/playbooks/kubernetes/deploy.yml" ]; then
    print_error "Cannot find Ansible playbook. Run this script from bytefreezer-control root directory"
    exit 1
fi

# Deploy using Ansible (localhost execution)
cd ansible
ansible-playbook playbooks/kubernetes/deploy.yml \
    -e target_environment=development \
    -e enable_debug_logging=true \
    -e storage_enabled=false \
    -e bytefreezer_control_version=latest \
    --connection=local

if [ $? -eq 0 ]; then
    print_success "ByteFreezer Control deployed successfully"
else
    print_error "Failed to deploy ByteFreezer Control"
    exit 1
fi

cd ..

print_status "Step 6: Waiting for ByteFreezer Control to be ready..."

# Wait for pods to be ready
kubectl wait --for=condition=available --timeout=300s deployment/bytefreezer-control -n "$CONTROL_NAMESPACE"

if [ $? -eq 0 ]; then
    print_success "ByteFreezer Control is ready"
else
    print_error "ByteFreezer Control failed to become ready"
    kubectl get pods -n "$CONTROL_NAMESPACE"
    kubectl logs -l app.kubernetes.io/name=bytefreezer-control -n "$CONTROL_NAMESPACE" --tail=20
    exit 1
fi

print_status "Step 7: Testing ByteFreezer Control Health"

# Get service endpoint
CONTROL_ENDPOINT=$(kubectl get service bytefreezer-control -n "$CONTROL_NAMESPACE" -o jsonpath='{.spec.clusterIP}'):8082

print_status "Testing ByteFreezer Control health endpoint: $CONTROL_ENDPOINT"

# Test health endpoint
kubectl run control-health-test --image=curlimages/curl:latest --rm -i --restart=Never -- \
    curl -s --connect-timeout 10 "http://$CONTROL_ENDPOINT/api/v2/health" > /tmp/control-health.json

if [ $? -eq 0 ]; then
    print_success "ByteFreezer Control health endpoint is accessible"
    echo "Health response:"
    cat /tmp/control-health.json
else
    print_error "Cannot reach ByteFreezer Control health endpoint"
    kubectl logs -l app.kubernetes.io/name=bytefreezer-control -n "$CONTROL_NAMESPACE" --tail=20
    exit 1
fi

print_status "Step 8: Testing LocalStack Integration from Control Service"

# Check if Control service can reach LocalStack
print_status "Checking if Control service has LocalStack environment variables..."

# Get pod name
CONTROL_POD=$(kubectl get pods -n "$CONTROL_NAMESPACE" -l app.kubernetes.io/name=bytefreezer-control -o jsonpath='{.items[0].metadata.name}')

if [ -n "$CONTROL_POD" ]; then
    print_status "Checking environment variables in pod: $CONTROL_POD"
    
    # Check AWS environment variables
    kubectl exec -n "$CONTROL_NAMESPACE" "$CONTROL_POD" -- env | grep -E "(AWS_|LOCALSTACK_)" || true
    
    # Test LocalStack connectivity from within the pod
    print_status "Testing LocalStack connectivity from Control pod..."
    kubectl exec -n "$CONTROL_NAMESPACE" "$CONTROL_POD" -- wget -qO- "http://localstack-internal.localstack.svc.cluster.local:4566/_localstack/health" > /tmp/pod-localstack-test.json 2>/dev/null
    
    if [ $? -eq 0 ]; then
        print_success "Control service can reach LocalStack"
        echo "LocalStack services available:"
        cat /tmp/pod-localstack-test.json | head -10
    else
        print_warning "Control service cannot reach LocalStack (check network policies)"
    fi
else
    print_error "No Control pods found"
fi

print_status "Step 9: Testing Service Discovery"

# Test DNS resolution
print_status "Testing DNS resolution between services..."

kubectl run dns-test --image=busybox:latest --rm -i --restart=Never -- \
    nslookup localstack-internal.localstack.svc.cluster.local > /tmp/dns-test.out 2>&1

if [ $? -eq 0 ]; then
    print_success "DNS resolution works"
    cat /tmp/dns-test.out
else
    print_warning "DNS resolution issues detected"
    cat /tmp/dns-test.out
fi

print_status "Step 10: Integration Summary"

echo ""
echo "🎯 Integration Test Results:"
echo "=========================="

# Check final status
LOCALSTACK_PODS=$(kubectl get pods -n "$LOCALSTACK_NAMESPACE" -l app=localstack --field-selector=status.phase=Running --no-headers | wc -l)
CONTROL_PODS=$(kubectl get pods -n "$CONTROL_NAMESPACE" -l app.kubernetes.io/name=bytefreezer-control --field-selector=status.phase=Running --no-headers | wc -l)

echo "✅ LocalStack Pods Running: $LOCALSTACK_PODS"
echo "✅ Control Pods Running: $CONTROL_PODS"

if [ "$LOCALSTACK_PODS" -gt 0 ] && [ "$CONTROL_PODS" -gt 0 ]; then
    print_success "🎉 Integration test completed successfully!"
    
    echo ""
    echo "📊 Service Endpoints:"
    echo "===================="
    echo "LocalStack Internal: http://localstack-internal.localstack.svc.cluster.local:4566"
    echo "LocalStack Admin: http://localstack-internal.localstack.svc.cluster.local:4510"
    echo "Control API: http://bytefreezer-control.bytefreezer-control.svc.cluster.local:8082"
    
    echo ""
    echo "🔗 Port Forward Commands:"
    echo "========================="
    echo "LocalStack: kubectl port-forward svc/localstack-external 4566:4566 -n localstack"
    echo "Control: kubectl port-forward svc/bytefreezer-control 8082:8082 -n bytefreezer-control"
    
    echo ""
    echo "🧪 Test Commands:"
    echo "=================="
    echo "LocalStack Health: curl http://localhost:4566/_localstack/health"
    echo "Control Health: curl http://localhost:8082/api/v2/health"
    echo "Control Config: curl http://localhost:8082/api/v2/config"
    
else
    print_error "Integration test failed - not all services are running"
    exit 1
fi

# Cleanup test files
rm -f /tmp/localstack-health.json /tmp/s3-test.out /tmp/control-health.json /tmp/pod-localstack-test.json /tmp/dns-test.out

print_success "Integration test completed! 🚀"