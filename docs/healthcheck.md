# Health Check Configuration

The XDS Control Plane Operator supports configuring health checks for Envoy clusters. Health checks help ensure that traffic is only routed to healthy endpoints.

## Overview

Health checks are configured at the cluster level and support three types:
- **HTTP Health Checks** - For HTTP/HTTPS services
- **TCP Health Checks** - For TCP services
- **gRPC Health Checks** - For gRPC services

## Configuration

Health checks are configured using the `healthCheck` field in the cluster specification:

```yaml
spec:
  clusters:
    - name: web-backend
      type: strict_dns
      lbPolicy: round_robin
      healthCheck:
        timeout: 3s
        interval: 10s
        intervalJitter: 1s
        unhealthyThreshold: 3
        healthyThreshold: 2
        reuseConnection: true
        # Health check type configuration
```

## Common Health Check Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `timeout` | duration | 5s | Time to wait for a health check response |
| `interval` | duration | 10s | Interval between health checks |
| `intervalJitter` | duration | - | Amount of jitter to add to the interval |
| `unhealthyThreshold` | int32 | 3 | Number of unhealthy checks before marking host as unhealthy |
| `healthyThreshold` | int32 | 2 | Number of healthy checks before marking host as healthy |
| `reuseConnection` | bool | false | Whether to reuse health check connections |

## HTTP Health Checks

HTTP health checks send HTTP requests to a specified path:

```yaml
healthCheck:
  timeout: 3s
  interval: 10s
  httpHealthCheck:
    path: /health
    host: web-service.local
    requestHeadersToAdd:
      - header:
          key: "X-Health-Check"
          value: "envoy"
        append: false
    expectedStatuses:
      - start: 200
        end: 299
      - start: 404
        end: 404
```

### HTTP Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `path` | string | HTTP path for health checks (required) |
| `host` | string | Host header value |
| `requestHeadersToAdd` | array | Additional headers to send |
| `expectedStatuses` | array | Expected HTTP status code ranges |

## TCP Health Checks

TCP health checks establish TCP connections and optionally send/receive data:

```yaml
healthCheck:
  timeout: 5s
  interval: 15s
  tcpHealthCheck:
    send: "PING"
    receive:
      - "PONG"
```

### TCP Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `send` | bytes | Data to send during TCP health check |
| `receive` | array of bytes | Expected response data |

## gRPC Health Checks

gRPC health checks use the gRPC health checking protocol:

```yaml
healthCheck:
  timeout: 2s
  interval: 8s
  grpcHealthCheck:
    serviceName: "myapp.v1.HealthService"
    authority: "grpc-service.local"
```

### gRPC Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `serviceName` | string | Service name for gRPC health checks |
| `authority` | string | Authority header value |

## Complete Examples

### Web Service with HTTP Health Check

```yaml
apiVersion: xds.okassov/v1alpha1
kind: XDSControlPlane
metadata:
  name: web-service-example
spec:
  xdsPort: 18000
  nodeIDs:
    - external-envoy
  listeners:
    - name: web_listener
      address: 0.0.0.0
      port: 80
      filterChains:
        - filters:
            - name: envoy.filters.network.tcp_proxy
              typedConfig:
                "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
                stat_prefix: web_proxy
                cluster: web-backend
  clusters:
    - name: web-backend
      type: strict_dns
      lbPolicy: round_robin
      connectTimeout: 5s
      loadAssignment:
        endpointsFrom:
          type: Node
          selector:
            matchLabels:
              node-role.kubernetes.io/worker: ""
          port: 30080
      healthCheck:
        timeout: 3s
        interval: 10s
        intervalJitter: 1s
        unhealthyThreshold: 3
        healthyThreshold: 2
        reuseConnection: true
        httpHealthCheck:
          path: /health
          host: web-service.local
          expectedStatuses:
            - start: 200
              end: 299
```

### Database with TCP Health Check

```yaml
apiVersion: xds.okassov/v1alpha1
kind: XDSControlPlane
metadata:
  name: database-example
spec:
  xdsPort: 18000
  nodeIDs:
    - external-envoy
  listeners:
    - name: database_listener
      address: 0.0.0.0
      port: 5432
      filterChains:
        - filters:
            - name: envoy.filters.network.tcp_proxy
              typedConfig:
                "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
                stat_prefix: db_proxy
                cluster: postgres-backend
  clusters:
    - name: postgres-backend
      type: strict_dns
      lbPolicy: round_robin
      connectTimeout: 5s
      loadAssignment:
        endpointsFrom:
          type: Node
          selector:
            matchLabels:
              node-role.kubernetes.io/worker: ""
          port: 30432
      healthCheck:
        timeout: 5s
        interval: 15s
        unhealthyThreshold: 5
        healthyThreshold: 3
        reuseConnection: false
        tcpHealthCheck:
          send: "SELECT 1;"
          receive:
            - "1"
```

### gRPC Service with gRPC Health Check

```yaml
apiVersion: xds.okassov/v1alpha1
kind: XDSControlPlane
metadata:
  name: grpc-service-example
spec:
  xdsPort: 18000
  nodeIDs:
    - external-envoy
  listeners:
    - name: grpc_listener
      address: 0.0.0.0
      port: 9090
      filterChains:
        - filters:
            - name: envoy.filters.network.tcp_proxy
              typedConfig:
                "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
                stat_prefix: grpc_proxy
                cluster: grpc-backend
  clusters:
    - name: grpc-backend
      type: strict_dns
      lbPolicy: least_request
      connectTimeout: 3s
      loadAssignment:
        endpointsFrom:
          type: Node
          selector:
            matchLabels:
              node-role.kubernetes.io/worker: ""
          port: 30090
      healthCheck:
        timeout: 2s
        interval: 8s
        intervalJitter: 500ms
        unhealthyThreshold: 2
        healthyThreshold: 1
        reuseConnection: true
        grpcHealthCheck:
          serviceName: "myapp.v1.HealthService"
          authority: "grpc-service.local"
```

## Default Behavior

If no specific health check type is configured (httpHealthCheck, tcpHealthCheck, or grpcHealthCheck), the system will default to a basic TCP health check that simply establishes a TCP connection to verify connectivity.

## Best Practices

1. **Choose appropriate intervals**: Don't make health checks too frequent to avoid overwhelming your services
2. **Set reasonable thresholds**: Balance between quick failure detection and avoiding false positives
3. **Use jitter**: Add jitter to prevent thundering herd problems when many health checkers start simultaneously
4. **Monitor health check metrics**: Envoy exposes metrics about health check results
5. **Test your health endpoints**: Ensure your application's health endpoints are lightweight and reliable 