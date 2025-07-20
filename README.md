# XDS Control Plane Operator

[![Build and Push Docker Image](https://github.com/okassov/xds-cp-operator/actions/workflows/docker-build-push.yml/badge.svg)](https://github.com/okassov/xds-cp-operator/actions/workflows/docker-build-push.yml)
[![Release](https://github.com/okassov/xds-cp-operator/actions/workflows/release.yml/badge.svg)](https://github.com/okassov/xds-cp-operator/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/okassov/xds-cp-operator)](https://goreportcard.com/report/github.com/okassov/xds-cp-operator)
[![Docker Pulls](https://img.shields.io/docker/pulls/okassov/xds-cp-operator)](https://hub.docker.com/r/okassov/xds-cp-operator)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/okassov/xds-cp-operator)](https://github.com/okassov/xds-cp-operator/releases)
[![License](https://img.shields.io/github/license/okassov/xds-cp-operator)](LICENSE)

Kubernetes operator for managing Envoy xDS control plane configurations with advanced health monitoring capabilities.

## ✨ Features

- **🏥 Advanced Health Checks**: HTTP, TCP, and gRPC health monitoring for Envoy clusters
- **🎯 Multiple Node Support**: Manage multiple Envoy proxy instances
- **📊 Comprehensive Status Tracking**: Real-time phases and conditions monitoring
- **🔄 Lifecycle Management**: Automatic xDS server lifecycle management
- **🔧 Universal Envoy Support**: Support for any Envoy configuration type
- **🔌 Transport Socket Support**: Proxy protocol, TLS, and raw buffer transport
- **📡 Real-time Configuration**: Live configuration updates via xDS protocol

## 🏥 Health Check Support

### Supported Health Check Types

#### HTTP Health Checks
- Custom paths and host headers
- Request header customization
- Expected status code ranges
- Configurable timeouts and intervals

#### TCP Health Checks  
- Binary payload support with Base64 encoding
- Send/receive data validation
- Connection reuse control

#### gRPC Health Checks
- Service name specification
- Authority header configuration
- Standard gRPC health checking protocol

### Health Check Configuration
```yaml
apiVersion: xds.okassov/v1alpha1
kind: XDSControlPlane
metadata:
  name: webapp-xds
spec:
  clusters:
    - name: web-backend
      type: strict_dns
      healthCheck:
        timeout: 3s
        interval: 5s
        intervalJitter: 1s
        unhealthyThreshold: 3
        healthyThreshold: 2
        httpHealthCheck:
          path: /health
          host: backend.local
          requestHeadersToAdd:
            - header:
                key: "X-Health-Check"
                value: "envoy-operator"
              append: false
          expectedStatuses:
            - start: 200
              end: 299
    
    - name: database-backend
      type: static
      healthCheck:
        timeout: 2s
        interval: 8s
        tcpHealthCheck:
          send: "U0VMRUNUIDEh"  # Base64: "SELECT 1!"
          receive:
            - "MQ=="  # Base64: "1"
```

## 🚀 Quick Start

### Option 1: Using Docker Image (Recommended)

```bash
# Install CRDs
kubectl apply -f https://github.com/okassov/xds-cp-operator/releases/latest/download/xds-cp-operator-crds.yaml

# Deploy operator
kubectl apply -f https://github.com/okassov/xds-cp-operator/releases/latest/download/xds-cp-operator.yaml

# Apply sample configuration
kubectl apply -f https://raw.githubusercontent.com/okassov/xds-cp-operator/main/config/samples/xds_v1alpha1_xdscontrolplane.yaml
```

### Option 2: Local Development

```bash
# Install CRDs
make install

# Run the operator locally
make run
```

### Verify Installation
```bash
# Check operator status
kubectl get pods -n xds-cp-operator-system

# Create and verify XDSControlPlane
kubectl apply -f config/samples/xds_v1alpha1_xdscontrolplane.yaml
kubectl get xdscontrolplane
kubectl describe xdscontrolplane example
```

### 🐳 Docker Images

Available on [Docker Hub](https://hub.docker.com/r/okassov/xds-cp-operator):

```bash
# Pull latest version
docker pull okassov/xds-cp-operator:latest

# Pull specific version
docker pull okassov/xds-cp-operator:v1.0.0

# Multi-platform support
docker pull okassov/xds-cp-operator:latest --platform linux/amd64
docker pull okassov/xds-cp-operator:latest --platform linux/arm64
```

## 📖 Configuration Examples

### Basic TCP Proxy
```yaml
apiVersion: xds.okassov/v1alpha1
kind: XDSControlPlane
metadata:
  name: tcp-proxy
spec:
  xdsPort: 18000
  nodeIDs:
    - envoy-proxy-1
  clusters:
    - name: backend-service
      type: strict_dns
      lbPolicy: round_robin
  listeners:
    - name: tcp-listener
      address: 0.0.0.0
      port: 8080
      filterChains:
        - filters:
            - name: envoy.filters.network.tcp_proxy
              typedConfig:
                "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
                cluster: backend-service
                stat_prefix: tcp_proxy
```

### Advanced Configuration with Health Checks
```yaml
apiVersion: xds.okassov/v1alpha1
kind: XDSControlPlane
metadata:
  name: advanced-setup
spec:
  xdsPort: 18000
  nodeIDs:
    - envoy-frontend
    - envoy-backend
  clusters:
    - name: api-service
      type: strict_dns
      lbPolicy: least_request
      connectTimeout: 5s
      healthCheck:
        timeout: 3s
        interval: 10s
        unhealthyThreshold: 3
        healthyThreshold: 2
        reuseConnection: true
        httpHealthCheck:
          path: /api/health
          host: api.service.local
          requestHeadersToAdd:
            - header:
                key: "User-Agent"
                value: "envoy-healthchecker/1.0"
      transportSocket:
        name: envoy.transport_sockets.tls
        typedConfig:
          "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
          sni: api.service.local
```

## 🔧 Supported Envoy Types

### Explicitly Optimized
- `envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy`
- `envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager`
- `envoy.extensions.transport_sockets.proxy_protocol.v3.ProxyProtocolUpstreamTransport`
- `envoy.extensions.transport_sockets.raw_buffer.v3.RawBuffer`
- `envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext`
- `envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext`
- `envoy.extensions.access_loggers.file.v3.FileAccessLog`
- `envoy.extensions.filters.listener.proxy_protocol.v3.ProxyProtocol`

### Universal Support
Any valid Envoy type URL (e.g., `type.googleapis.com/envoy.extensions.*`) is automatically supported through the universal fallback mechanism.

## 📚 Documentation

- **[Health Check Guide](docs/healthcheck.md)** - Complete health check configuration guide
- **[Integration Testing](docs/integration-testing.md)** - Testing with real Envoy proxies
- **Configuration Samples** - See `config/samples/` directory

## 🛠️ Development

### Prerequisites
- Go 1.19+
- Docker
- kubectl
- Kubernetes cluster (local or remote)

### Build and Test
```bash
# Generate code and manifests
make generate
make manifests

# Run tests
make test

# Build binary
make build

# Build and push Docker image
make docker-build docker-push IMG=your-registry/xds-cp-operator:tag
```

### Local Development
```bash
# Install CRDs
make install

# Run operator locally
make run

# Deploy sample configurations
kubectl apply -f config/samples/
```

## 🔍 Monitoring and Troubleshooting

### Check Operator Status
```bash
# View operator logs
kubectl logs -l app=xds-cp-operator -f

# Check XDSControlPlane status
kubectl get xdscontrolplane -o wide
kubectl describe xdscontrolplane <name>
```

### Health Check Validation
With a running Envoy proxy connected to the operator:
```bash
# Check cluster configuration
curl -s http://envoy-admin:9901/config_dump | jq '.configs[1].dynamic_active_clusters'

# View health check statistics  
curl -s http://envoy-admin:9901/stats | grep health_check
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📄 License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## 🏷️ Version

Current version: v1.0.0

**Features:**
- ✅ Full xDS v3 API support
- ✅ HTTP/TCP/gRPC health checks
- ✅ Real Envoy proxy integration tested
- ✅ Production-ready operator patterns
- ✅ Comprehensive documentation


## Helm Installation

### Add Helm Repository

```bash
helm repo add xds-cp-operator https://okassov.github.io/xds-cp-operator/
helm repo update
```

### Install the Operator

```bash
# Latest stable version
helm install xds-cp-operator xds-cp-operator/xds-cp-operator \
  --namespace xds-system \
  --create-namespace

# Specific version (1.2.1)
helm install xds-cp-operator xds-cp-operator/xds-cp-operator \
  --namespace xds-system \
  --create-namespace \
  --version 1.2.1

# With custom values
helm install xds-cp-operator xds-cp-operator/xds-cp-operator \
  --namespace xds-system \
  --create-namespace \
  --set image.tag=1.2.1 \
  --set xdsService.type=LoadBalancer
```

### Available Versions

```bash
helm search repo xds-cp-operator/xds-cp-operator --versions
```

