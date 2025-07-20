# XDS Control Plane Operator Helm Chart

This Helm chart installs the XDS Control Plane Operator for Kubernetes, which manages Envoy xDS control planes through custom resources.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+

## Installation

### Add Helm Repository (if published)

```bash
helm repo add xds-cp-operator https://okassov.github.io/xds-cp-operator
helm repo update
```

### Install from Local Chart

```bash
# Clone the repository
git clone https://github.com/okassov/xds-cp-operator.git
cd xds-cp-operator

# Install the chart
helm install xds-cp-operator deploy/chart/ \
  --namespace xds-system \
  --create-namespace
```

### Install with Custom Values

```bash
helm install xds-cp-operator deploy/chart/ \
  --namespace xds-system \
  --create-namespace \
  --set image.tag=v1.0.0 \
  --set resources.requests.memory=128Mi
```

## Configuration

The following table lists the configurable parameters and their default values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Operator image repository | `okassov/xds-cp-operator` |
| `image.tag` | Operator image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `replicaCount` | Number of operator replicas | `1` |
| `resources.requests.cpu` | CPU resource requests | `100m` |
| `resources.requests.memory` | Memory resource requests | `64Mi` |
| `resources.limits.cpu` | CPU resource limits | `500m` |
| `resources.limits.memory` | Memory resource limits | `512Mi` |
| `serviceAccount.create` | Create service account | `true` |
| `rbac.create` | Create RBAC resources | `true` |
| `operator.metricsAddr` | Metrics server address | `:8082` |
| `operator.enableLeaderElection` | Enable leader election | `true` |
| `service.type` | Service type for metrics | `ClusterIP` |
| `service.port` | Service port for metrics | `8082` |
| `xdsService.enabled` | Enable xDS service for external Envoy proxies | `true` |
| `xdsService.type` | xDS service type (ClusterIP/NodePort/LoadBalancer) | `ClusterIP` |
| `xdsService.portRange.start` | Start of xDS port range | `18000` |
| `xdsService.portRange.end` | End of xDS port range | `18010` |
| `serviceMonitor.enabled` | Enable Prometheus ServiceMonitor | `false` |
| `autoscaling.enabled` | Enable HPA | `false` |

## Examples

### Basic Installation

```bash
helm install xds-cp-operator deploy/chart/ \
  --namespace xds-system \
  --create-namespace
```

### External Envoy Access with NodePort

For environments where external Envoy proxies need to connect to the xDS servers:

```bash
helm install xds-cp-operator deploy/chart/ \
  --namespace xds-system \
  --create-namespace \
  --set xdsService.type=NodePort \
  --set xdsService.nodePortRange.start=30180 \
  --set xdsService.nodePortRange.end=30190
```

### External Envoy Access with LoadBalancer

For cloud environments with LoadBalancer support:

```bash
helm install xds-cp-operator deploy/chart/ \
  --namespace xds-system \
  --create-namespace \
  --set xdsService.type=LoadBalancer \
  --set xdsService.loadBalancerSourceRanges[0]="10.0.0.0/8" \
  --set xdsService.loadBalancerSourceRanges[1]="172.16.0.0/12"
```

### Production Installation with Monitoring

```bash
helm install xds-cp-operator deploy/chart/ \
  --namespace xds-system \
  --create-namespace \
  --values - <<EOF
replicaCount: 2

resources:
  requests:
    memory: 128Mi
    cpu: 200m
  limits:
    memory: 1Gi
    cpu: 1000m

# External xDS access configuration
xdsService:
  enabled: true
  type: LoadBalancer
  portRange:
    start: 18000
    end: 18020
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
  loadBalancerSourceRanges:
    - "10.0.0.0/8"

serviceMonitor:
  enabled: true
  namespace: monitoring

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 80

nodeSelector:
  kubernetes.io/os: linux

tolerations:
- key: node-role.kubernetes.io/control-plane
  effect: NoSchedule
EOF
```

### Install with Custom Image

```bash
helm install xds-cp-operator deploy/chart/ \
  --namespace xds-system \
  --create-namespace \
  --set image.repository=your-registry.com/xds-cp-operator \
  --set image.tag=v1.0.0 \
  --set image.pullPolicy=IfNotPresent
```

## Creating XDS Control Planes

After installing the operator, you can create XDS control plane instances:

```yaml
apiVersion: xds.okassov/v1alpha1
kind: XDSControlPlane
metadata:
  name: example-xds
  namespace: default
spec:
  xdsPort: 18000
  nodeIDs:
    - "envoy-proxy-1"
    - "envoy-proxy-2"
  clusters:
    - name: backend-service
      type: static
      lbPolicy: round_robin
      connectTimeout: 5s
      healthCheck:
        timeout: 5s
        interval: 10s
        unhealthyThreshold: 3
        healthyThreshold: 2
        httpHealthCheck:
          path: /health
          expectedStatuses:
            - start: 200
              end: 299
  listeners:
    - name: main-listener
      address: 0.0.0.0
      port: 8080
      filterChains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typedConfig:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: backend
                      domains: ["*"]
                      routes:
                        - match: { prefix: "/" }
                          route: { cluster: backend-service }
```

## External Envoy Configuration

When using external Envoy proxies, configure them to connect to the xDS service:

### NodePort Example

```yaml
# Get the node port
kubectl get svc xds-cp-operator-xds -n xds-system

# Envoy configuration
node:
  id: envoy-proxy-1
  cluster: my-cluster

dynamic_resources:
  ads_config:
    api_type: GRPC
    transport_api_version: V3
    grpc_services:
    - envoy_grpc:
        cluster_name: xds_cluster

  cds_config:
    resource_api_version: V3
    ads: {}

  lds_config:
    resource_api_version: V3
    ads: {}

static_resources:
  clusters:
  - name: xds_cluster
    type: STRICT_DNS
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        explicit_http_config:
          http2_protocol_options: {}
    load_assignment:
      cluster_name: xds_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: <NODE_IP>
                port_value: <NODE_PORT>  # e.g., 30180
```

### LoadBalancer Example

```yaml
# Envoy configuration for LoadBalancer
static_resources:
  clusters:
  - name: xds_cluster
    type: STRICT_DNS
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        explicit_http_config:
          http2_protocol_options: {}
    load_assignment:
      cluster_name: xds_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: <LOADBALANCER_IP>
                port_value: 18000
```

## Upgrading

```bash
helm upgrade xds-cp-operator deploy/chart/ \
  --namespace xds-system
```

## Uninstalling

```bash
helm uninstall xds-cp-operator --namespace xds-system
```

**Note:** This will remove the operator but not the CRDs. To remove CRDs:

```bash
kubectl delete crd xdscontrolplanes.xds.okassov
```

## Health Checks Support

The operator supports comprehensive health check configurations:

- **HTTP Health Checks**: Custom paths, headers, expected status codes
- **TCP Health Checks**: Binary payloads for send/receive validation  
- **gRPC Health Checks**: Service name and authority configuration

See the main repository README for detailed health check examples.

## Monitoring

When `serviceMonitor.enabled=true`, the chart creates a ServiceMonitor resource for Prometheus to scrape operator metrics including:

- Reconciliation metrics
- Error rates
- Performance metrics
- Custom operator metrics

## Networking Considerations

### Port Management

The operator automatically manages xDS server ports based on the `XDSControlPlane` specifications. The `xdsService.portRange` defines which ports are exposed through Kubernetes services.

### Security

- Use `loadBalancerSourceRanges` to restrict access to trusted networks
- Consider using internal load balancers for security
- Implement network policies for additional security

### Performance

- Use LoadBalancer type for production external access
- Consider using multiple operator replicas with leader election
- Monitor xDS connection metrics

## Troubleshooting

### Check operator status
```bash
kubectl get deployment xds-cp-operator -n xds-system
kubectl logs -f deployment/xds-cp-operator -n xds-system
```

### Check xDS services
```bash
kubectl get svc -n xds-system
kubectl get xdscontrolplane -o wide
```

### Test xDS connectivity
```bash
# Port forward for testing
kubectl port-forward svc/xds-cp-operator-xds 18000:18000 -n xds-system

# Test with grpcurl
grpcurl -plaintext localhost:18000 envoy.service.discovery.v3.AggregatedDiscoveryService/StreamAggregatedResources
```

## Support

- [GitHub Issues](https://github.com/okassov/xds-cp-operator/issues)
- [Documentation](https://github.com/okassov/xds-cp-operator)

## License

This project is licensed under the Apache License 2.0. 
