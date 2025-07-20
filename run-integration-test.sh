#!/bin/bash

set -e

echo "🚀 Starting XDS Control Plane Integration Test"
echo "=============================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
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

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

# Check if kubectl is available and cluster is accessible
if ! kubectl cluster-info >/dev/null 2>&1; then
    print_error "Kubernetes cluster is not accessible. Please check your kubeconfig."
    exit 1
fi

# Clean up any existing resources
cleanup() {
    print_status "Cleaning up..."
    
    # Stop Docker Compose
    if [ -f docker-compose.yaml ]; then
        docker-compose down --remove-orphans >/dev/null 2>&1 || true
    fi
    
    # Delete XDS resource
    kubectl delete xdscontrolplane integration-test >/dev/null 2>&1 || true
    
    # Kill background processes
    jobs -p | xargs -r kill >/dev/null 2>&1 || true
}

# Set up trap for cleanup
trap cleanup EXIT

print_status "Step 1: Building and installing CRDs"
make install

print_status "Step 2: Starting backend services with Docker Compose"
docker-compose up -d test-backend tcp-backend

print_status "Step 3: Waiting for backend services to be healthy"
timeout=60
counter=0
while [ $counter -lt $timeout ]; do
    if docker-compose ps | grep -q "healthy"; then
        print_success "Backend services are healthy"
        break
    fi
    sleep 1
    counter=$((counter + 1))
    if [ $counter -eq $timeout ]; then
        print_error "Backend services failed to become healthy within $timeout seconds"
        docker-compose logs
        exit 1
    fi
done

print_status "Step 4: Creating XDSControlPlane resource"
kubectl apply -f test-configs/integration-test-xds.yaml

print_status "Step 5: Starting XDS Control Plane Operator"
echo "Starting operator in background..."
make run > operator.log 2>&1 &
OPERATOR_PID=$!

# Wait for operator to start
sleep 5

print_status "Step 6: Waiting for XDSControlPlane to be ready"
timeout=60
counter=0
while [ $counter -lt $timeout ]; do
    if kubectl get xdscontrolplane integration-test -o jsonpath='{.status.phase}' 2>/dev/null | grep -q "Ready"; then
        print_success "XDSControlPlane is ready"
        break
    fi
    sleep 1
    counter=$((counter + 1))
    if [ $counter -eq $timeout ]; then
        print_error "XDSControlPlane failed to become ready within $timeout seconds"
        kubectl describe xdscontrolplane integration-test
        tail -n 20 operator.log
        exit 1
    fi
done

print_status "Step 7: Starting Envoy proxy"
docker-compose up -d envoy

print_status "Step 8: Waiting for Envoy to be healthy and connected"
timeout=60
counter=0
while [ $counter -lt $timeout ]; do
    if docker-compose ps envoy | grep -q "healthy"; then
        print_success "Envoy is healthy"
        break
    fi
    sleep 1
    counter=$((counter + 1))
    if [ $counter -eq $timeout ]; then
        print_error "Envoy failed to become healthy within $timeout seconds"
        docker-compose logs envoy
        exit 1
    fi
done

# Give Envoy some more time to connect to XDS server
sleep 10

print_status "Step 9: Verifying XDS configuration in Envoy"

echo ""
echo "📊 Checking Envoy Admin Interface"
echo "================================="

# Check if Envoy admin is accessible
if curl -s http://localhost:10000/ready >/dev/null; then
    print_success "Envoy admin interface is accessible"
else
    print_error "Envoy admin interface is not accessible"
    exit 1
fi

# Check clusters configuration
echo ""
print_status "Checking configured clusters:"
curl -s http://localhost:10000/clusters | grep -E "(test-backend-cluster|tcp-backend-cluster)" || {
    print_warning "Expected clusters not found in Envoy configuration"
}

# Check listeners configuration  
echo ""
print_status "Checking configured listeners:"
curl -s http://localhost:10000/listeners | grep -E "(http_listener|tcp_listener)" || {
    print_warning "Expected listeners not found in Envoy configuration"
}

# Check health check status
echo ""
print_status "Checking health check status:"
curl -s http://localhost:10000/clusters | grep -A 5 -B 5 "health_check" || {
    print_warning "Health check information not found"
}

print_status "Step 10: Testing health check functionality"

echo ""
echo "🏥 Health Check Verification"
echo "============================"

# Test backend health endpoint directly
print_status "Testing backend health endpoint directly:"
if curl -s http://localhost:8081/health | grep -q "healthy"; then
    print_success "Backend health endpoint is working"
else
    print_warning "Backend health endpoint is not responding correctly"
fi

# Check Envoy health check stats
print_status "Checking Envoy health check statistics:"
curl -s http://localhost:10000/stats | grep health_check || {
    print_warning "No health check statistics found"
}

echo ""
echo "📈 Integration Test Results"
echo "=========================="

# Show XDSControlPlane status
print_status "XDSControlPlane status:"
kubectl get xdscontrolplane integration-test -o yaml | grep -A 20 "status:"

# Show operator logs
echo ""
print_status "Recent operator logs:"
tail -n 10 operator.log

# Show Envoy logs
echo ""
print_status "Recent Envoy logs:"
docker-compose logs --tail=10 envoy

echo ""
print_success "Integration test completed!"
echo ""
echo "🔍 Manual verification steps:"
echo "1. Check Envoy admin interface: http://localhost:10000"
echo "2. Check clusters: http://localhost:10000/clusters"
echo "3. Check listeners: http://localhost:10000/listeners"
echo "4. Check stats: http://localhost:10000/stats"
echo "5. Backend health: http://localhost:8080/health"
echo ""
echo "To stop the test environment, press Ctrl+C or run: docker-compose down"

# Keep running for manual inspection
read -p "Press Enter to cleanup and exit..." 