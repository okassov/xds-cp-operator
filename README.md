# XDS Control Plane Operator

Kubernetes operator for managing Envoy xDS control plane configurations.

## Features

- Multiple Envoy node ID support
- Comprehensive status tracking with phases and conditions  
- Lifecycle management for xDS servers
- Universal Envoy type support through fallback mechanism
- Proxy protocol transport socket support

## Supported Envoy Types

### Explicitly Supported (optimized):
- `envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy`
- `envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager`
- `envoy.extensions.transport_sockets.proxy_protocol.v3.ProxyProtocolUpstreamTransport`
- `envoy.extensions.transport_sockets.raw_buffer.v3.RawBuffer`
- `envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext`
- `envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext`

### Universal Support:
Any valid Envoy type URL (e.g., `type.googleapis.com/envoy.extensions.*`) is automatically supported through the universal fallback mechanism.

## Quick Start

1. Install the operator:
```bash
make install
make run
```

2. Apply sample configuration:
```bash
kubectl apply -f config/samples/xds_v1alpha1_xdscontrolplane.yaml
```

## Configuration Examples

### Basic TCP Proxy with Proxy Protocol:
```yaml
apiVersion: xds.okassov/v1alpha1
kind: XDSControlPlane
metadata:
  name: example
spec:
  xdsPort: 18000
  nodeIDs:
    - envoy-1
    - envoy-2
  clusters:
    - name: backend
      type: strict_dns
      transportSocket:
        name: envoy.transport_sockets.upstream_proxy_protocol
        typedConfig:
          "@type": type.googleapis.com/envoy.extensions.transport_sockets.proxy_protocol.v3.ProxyProtocolUpstreamTransport
          config:
            version: V1
          transport_socket:
            name: envoy.transport_sockets.raw_buffer
            typedConfig:
              "@type": type.googleapis.com/envoy.extensions.transport_sockets.raw_buffer.v3.RawBuffer
```

## Development

```bash
make build
make test
make generate
make manifests
```

