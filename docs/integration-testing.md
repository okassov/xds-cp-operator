# Integration Testing with Envoy Proxy

This document describes how to run comprehensive integration tests for the XDS Control Plane Operator using a real Envoy proxy instance.

## Overview

The integration testing setup includes:

- **Real Envoy proxy** running in Docker
- **Backend test services** (HTTP and TCP) 
- **XDS Control Plane Operator** connecting to Kubernetes
- **Health check validation** end-to-end

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Envoy Proxy   │◄──►│ XDS Control Plane│◄──►│   Kubernetes    │
│   (Docker)      │    │    Operator      │    │    Cluster      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌──────────────────┐
│ Backend Services│    │ Health Checks    │
│   (Docker)      │    │  Verification    │
└─────────────────┘    └──────────────────┘
```

## Prerequisites

### Required Software

- **Docker** and **Docker Compose**
- **Kubernetes cluster** with kubectl access
- **Go 1.21+** for building the operator
- **Make** for build automation
- **curl** for HTTP testing

### Port Requirements

The following ports must be available on your local machine:

- `8081` - Test backend HTTP service
- `9091` - Test backend TCP service  
- `10000` - Envoy admin interface
- `8000` - Envoy HTTP proxy port
- `8001` - Envoy TCP proxy port
- `18003` - XDS Control Plane server

## Test Components

### 1. Backend Services

#### HTTP Backend (nginx)
- **Image**: `nginx:alpine`
- **Port**: `8080`
- **Endpoints**:
  - `/health` - Health check endpoint
  - `/api/health` - JSON health check endpoint
  - `/test` - Test endpoint for requests

#### TCP Backend (Python)
- **Image**: `python:3.9-alpine`
- **Port**: `9090`
- **Functionality**: Simple TCP server that responds to health check queries

### 2. Envoy Proxy

- **Image**: `envoyproxy/envoy:v1.28-latest`
- **Configuration**: Dynamic via XDS from our operator
- **Node ID**: `test-envoy-node`
- **Admin Interface**: `http://localhost:10000`

### 3. XDS Control Plane

The test creates an `XDSControlPlane` resource with:

- **HTTP Health Checks** for the nginx backend
- **TCP Health Checks** for the TCP backend
- **Custom headers** and **status code validation**
- **Different health check intervals** and **thresholds**

## Running Integration Tests

### Quick Start

Run the complete integration test with a single command:

```bash
./run-integration-test.sh
```

This script will:

1. ✅ Build and install CRDs
2. ✅ Start backend services  
3. ✅ Create XDSControlPlane resource
4. ✅ Start the operator
5. ✅ Launch Envoy proxy
6. ✅ Verify configurations
7. ✅ Test health checks
8. ✅ Display results

### Manual Step-by-Step

If you prefer to run tests manually:

#### 1. Install CRDs
```bash
make install
```

#### 2. Start Backend Services
```bash
docker-compose up -d test-backend tcp-backend
```

#### 3. Verify Backend Health
```bash
# Check HTTP backend
curl http://localhost:8081/health

# Check TCP backend  
echo "test" | nc localhost 9091
```

#### 4. Create XDS Resource
```bash
kubectl apply -f test-configs/integration-test-xds.yaml
```

#### 5. Start Operator
```bash
make run
```

#### 6. Start Envoy
```bash
docker-compose up -d envoy
```

#### 7. Verify Integration
```bash
# Check Envoy admin
curl http://localhost:10000/ready

# Check clusters
curl http://localhost:10000/clusters

# Check health check stats
curl http://localhost:10000/stats | grep health_check
```

## Verification Steps

### 1. XDS Configuration Delivery

Verify that Envoy received the configuration from our XDS server:

```bash
# Check configured clusters
curl -s http://localhost:10000/clusters | grep -E "(test-backend-cluster|tcp-backend-cluster)"

# Check configured listeners  
curl -s http://localhost:10000/listeners | grep -E "(http_listener|tcp_listener)"

# Check dynamic configuration source
curl -s http://localhost:10000/config_dump | jq '.configs[] | select(.["@type"] | contains("Cluster"))'
```

### 2. Health Check Functionality

Verify that health checks are working:

```bash
# Check health check statistics
curl -s http://localhost:10000/stats | grep -E "health_check\.(attempt|success|failure)"

# Check cluster health status
curl -s http://localhost:10000/clusters | grep -A 10 -B 5 "health_check"

# Monitor health check logs
docker-compose logs -f envoy | grep -i health
```

### 3. XDSControlPlane Status

Verify the operator is working correctly:

```bash
# Check resource status
kubectl get xdscontrolplane integration-test -o yaml

# Check operator logs for health check processing
kubectl logs -l app=xds-cp-operator | grep -i "health check"
```

## Expected Results

### Successful Integration

When the integration test passes, you should see:

✅ **XDSControlPlane Status**: `Ready`
```yaml
status:
  phase: Ready
  conditions:
  - type: Ready
    status: "True"
  - type: ServerUp  
    status: "True"
  - type: SnapshotReady
    status: "True"
```

✅ **Envoy Configuration**: Clusters with health checks
```
test-backend-cluster::health_check_filter::health_check_request::success: 5
test-backend-cluster::health_check_filter::health_check_request::total: 5
tcp-backend-cluster::health_check_filter::health_check_request::success: 3
tcp-backend-cluster::health_check_filter::health_check_request::total: 3
```

✅ **Backend Services**: Responding to health checks
```
nginx logs: "GET /health HTTP/1.1" 200
tcp-backend logs: Received: SELECT 1!
```

### Health Check Configuration

The test validates these health check features:

#### HTTP Health Checks
- ✅ Custom path: `/health`
- ✅ Custom headers: `X-Health-Check: envoy-integration-test`
- ✅ Expected status codes: `200-299`
- ✅ Connection reuse: `true`
- ✅ Timeout: `3s`, Interval: `5s`

#### TCP Health Checks  
- ✅ Send data: `SELECT 1!` (base64 encoded)
- ✅ Receive data: `1` (base64 encoded)
- ✅ Connection reuse: `false`
- ✅ Timeout: `2s`, Interval: `8s`

## Troubleshooting

### Common Issues

#### 1. Envoy Can't Connect to XDS Server

**Symptoms**: Envoy logs show connection refused to `host.docker.internal:18003`

**Solutions**:
- Ensure the operator is running on port 18003
- Check firewall settings
- On Linux, replace `host.docker.internal`