package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	clustergrpc "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointgrpc "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenergrpc "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routegrpc "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
	ctrlLog "sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/okassov/xds-cp-operator/api/v1alpha1"

	types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	res "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoytype "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	// Transport sockets
	proxy_protocol "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/proxy_protocol/v3"
	raw_buffer "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/raw_buffer/v3"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	// Network filters
	http_connection_manager "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"

	// Access loggers
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	file_access_log "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	listener_proxy_protocol "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/proxy_protocol/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"sync"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type XDSControlPlaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// XDSServerManager manages the lifecycle of xDS servers
type XDSServerManager struct {
	sync.RWMutex
	servers map[string]*XDSServerInstance
}

type XDSServerInstance struct {
	server   *grpc.Server
	cache    cache.SnapshotCache
	listener net.Listener
	cancel   context.CancelFunc
	port     int
}

var (
	serverManager = &XDSServerManager{
		servers: make(map[string]*XDSServerInstance),
	}
	cacheInitOnce sync.Once
)

const (
	XDSControlPlaneFinalizer = "xds.okassov/finalizer"

	// Condition types
	ConditionTypeReady    = "Ready"
	ConditionTypeServerUp = "ServerUp"
	ConditionTypeSnapshot = "SnapshotReady"

	// Phase values
	PhasePending = "Pending"
	PhaseReady   = "Ready"
	PhaseError   = "Error"
)

func (r *XDSControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.XDSControlPlane{}).
		Complete(r)
}

func (r *XDSControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrlLog.FromContext(ctx).WithValues("xdscontrolplane", req.NamespacedName)
	log.Info("Reconciling XDSControlPlane", "request", req)
	defer log.Info("Finished reconciling XDSControlPlane", "request", req)

	var xdsCRD api.XDSControlPlane
	if err := r.Get(ctx, req.NamespacedName, &xdsCRD); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("XDSControlPlane not found, cleaning up server")
			r.cleanupServer(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch XDSControlPlane")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !xdsCRD.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &xdsCRD)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&xdsCRD, XDSControlPlaneFinalizer) {
		controllerutil.AddFinalizer(&xdsCRD, XDSControlPlaneFinalizer)
		return ctrl.Result{}, r.Update(ctx, &xdsCRD)
	}

	// Update status to Pending initially
	if xdsCRD.Status.Phase == "" {
		return r.updateStatus(ctx, &xdsCRD, PhasePending, "Initializing xDS control plane")
	}

	// Ensure xDS server is running
	serverKey := req.NamespacedName.String()
	server, err := r.ensureXDSServer(ctx, &xdsCRD, serverKey)
	if err != nil {
		log.Error(err, "Failed to ensure xDS server")
		r.updateStatus(ctx, &xdsCRD, PhaseError, fmt.Sprintf("Failed to start xDS server: %v", err))
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	// Build and set snapshot
	snapshot, err := r.buildXDSSnapshot(ctx, &xdsCRD)
	if err != nil {
		log.Error(err, "Error building xDS snapshot")
		r.updateStatus(ctx, &xdsCRD, PhaseError, fmt.Sprintf("Failed to build snapshot: %v", err))
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	// Get nodeIDs to use
	nodeIDs := xdsCRD.Spec.NodeIDs
	if len(nodeIDs) == 0 {
		nodeIDs = []string{"external-envoy"}
	}

	// Set snapshot for all nodeIDs
	version := strconv.FormatInt(time.Now().Unix(), 10)
	for _, nodeID := range nodeIDs {
		log.Info("Setting xDS snapshot", "nodeID", nodeID, "version", version)
		if err := server.cache.SetSnapshot(ctx, nodeID, &snapshot); err != nil {
			log.Error(err, "failed to set xDS snapshot", "nodeID", nodeID)
			r.updateStatus(ctx, &xdsCRD, PhaseError, fmt.Sprintf("Failed to set snapshot for node %s: %v", nodeID, err))
			return ctrl.Result{RequeueAfter: time.Second * 30}, err
		}
	}

	log.Info("Successfully set xDS snapshots", "nodeIDs", nodeIDs, "version", version)

	// Update status to Ready
	return r.updateStatusReady(ctx, &xdsCRD, nodeIDs, server.port, version)
}

func (r *XDSControlPlaneReconciler) ensureXDSServer(ctx context.Context, crd *api.XDSControlPlane, serverKey string) (*XDSServerInstance, error) {
	log := ctrlLog.FromContext(ctx).WithValues("xdscontrolplane", crd.Name)

	serverManager.Lock()
	defer serverManager.Unlock()

	// Check if server already exists
	if server, exists := serverManager.servers[serverKey]; exists {
		if server.port == crd.Spec.XdsPort {
			log.Info("xDS server already running", "port", server.port)
			return server, nil
		}
		// Port changed, need to restart server
		log.Info("xDS server port changed, restarting", "oldPort", server.port, "newPort", crd.Spec.XdsPort)
		r.stopServer(server)
		delete(serverManager.servers, serverKey)
	}

	// Start new server
	server, err := r.startXDSServer(ctx, crd)
	if err != nil {
		return nil, err
	}

	serverManager.servers[serverKey] = server
	log.Info("xDS server started successfully", "port", server.port)
	return server, nil
}

func (r *XDSControlPlaneReconciler) startXDSServer(ctx context.Context, crd *api.XDSControlPlane) (*XDSServerInstance, error) {
	log := ctrlLog.FromContext(ctx).WithValues("xdscontrolplane", crd.Name)

	port := crd.Spec.XdsPort
	if port == 0 {
		port = 18000
	}

	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	log.Info("xDS gRPC server listening", "addr", addr)

	// Create snapshot cache and server
	snapCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	srv := grpc.NewServer()

	serverCtx, cancel := context.WithCancel(ctx)
	xdsServer := serverv3.NewServer(serverCtx, snapCache, nil)

	// Register all xDS services
	log.Info("Registering xDS gRPC services")
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(srv, xdsServer)
	endpointgrpc.RegisterEndpointDiscoveryServiceServer(srv, xdsServer)
	clustergrpc.RegisterClusterDiscoveryServiceServer(srv, xdsServer)
	listenergrpc.RegisterListenerDiscoveryServiceServer(srv, xdsServer)
	routegrpc.RegisterRouteDiscoveryServiceServer(srv, xdsServer)

	// Start serving
	go func() {
		log.Info("xDS gRPC server serving")
		if err := srv.Serve(lis); err != nil {
			log.Error(err, "xDS gRPC server exited with error")
		}
	}()

	return &XDSServerInstance{
		server:   srv,
		cache:    snapCache,
		listener: lis,
		cancel:   cancel,
		port:     port,
	}, nil
}

func (r *XDSControlPlaneReconciler) handleDeletion(ctx context.Context, crd *api.XDSControlPlane) (ctrl.Result, error) {
	log := ctrlLog.FromContext(ctx).WithValues("xdscontrolplane", crd.Name)
	log.Info("Handling deletion of XDSControlPlane")

	// Clean up server
	serverKey := fmt.Sprintf("%s/%s", crd.Namespace, crd.Name)
	r.cleanupServer(serverKey)

	// Remove finalizer
	controllerutil.RemoveFinalizer(crd, XDSControlPlaneFinalizer)
	return ctrl.Result{}, r.Update(ctx, crd)
}

func (r *XDSControlPlaneReconciler) cleanupServer(serverKey string) {
	serverManager.Lock()
	defer serverManager.Unlock()

	if server, exists := serverManager.servers[serverKey]; exists {
		r.stopServer(server)
		delete(serverManager.servers, serverKey)
	}
}

func (r *XDSControlPlaneReconciler) stopServer(server *XDSServerInstance) {
	if server.cancel != nil {
		server.cancel()
	}
	if server.server != nil {
		server.server.GracefulStop()
	}
	if server.listener != nil {
		server.listener.Close()
	}
}

func (r *XDSControlPlaneReconciler) updateStatus(ctx context.Context, crd *api.XDSControlPlane, phase, message string) (ctrl.Result, error) {
	crd.Status.Phase = phase

	condition := metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             phase,
		Message:            message,
	}

	if phase == PhaseReady {
		condition.Status = metav1.ConditionTrue
	}

	meta.SetStatusCondition(&crd.Status.Conditions, condition)

	return ctrl.Result{}, r.Status().Update(ctx, crd)
}

func (r *XDSControlPlaneReconciler) updateStatusReady(ctx context.Context, crd *api.XDSControlPlane, nodeIDs []string, port int, version string) (ctrl.Result, error) {
	crd.Status.Phase = PhaseReady
	crd.Status.ConnectedNodeIDs = nodeIDs
	crd.Status.XdsServerAddress = fmt.Sprintf(":%d", port)
	crd.Status.LastSnapshotVersion = version

	// Set conditions
	readyCondition := metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             "Ready",
		Message:            "XDS control plane is ready and serving configuration",
	}

	serverCondition := metav1.Condition{
		Type:               ConditionTypeServerUp,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             "ServerRunning",
		Message:            fmt.Sprintf("xDS server is running on port %d", port),
	}

	snapshotCondition := metav1.Condition{
		Type:               ConditionTypeSnapshot,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             "SnapshotReady",
		Message:            fmt.Sprintf("Snapshot version %s set for %d node(s)", version, len(nodeIDs)),
	}

	meta.SetStatusCondition(&crd.Status.Conditions, readyCondition)
	meta.SetStatusCondition(&crd.Status.Conditions, serverCondition)
	meta.SetStatusCondition(&crd.Status.Conditions, snapshotCondition)

	return ctrl.Result{}, r.Status().Update(ctx, crd)
}

func (r *XDSControlPlaneReconciler) buildXDSSnapshot(ctx context.Context, crd *api.XDSControlPlane) (cache.Snapshot, error) {
	log := ctrlLog.FromContext(ctx).WithValues("xdscontrolplane", crd.Name)
	log.Info("Building xDS snapshot", "crd", crd.Name)
	defer log.Info("Finished building xDS snapshot", "crd", crd.Name)

	var endpoints []types.Resource
	var clusters []types.Resource
	var listeners []types.Resource

	// Build clusters and endpoints
	for _, c := range crd.Spec.Clusters {
		log := log.WithValues("cluster", c.Name)
		log.Info("Processing cluster", "spec", c)

		clusterObj, cla, err := r.buildCluster(ctx, c)
		if err != nil {
			return cache.Snapshot{}, fmt.Errorf("failed to build cluster %s: %w", c.Name, err)
		}

		clusters = append(clusters, clusterObj)
		if cla != nil {
			endpoints = append(endpoints, cla)
		}
	}

	// Build listeners
	for _, l := range crd.Spec.Listeners {
		log := log.WithValues("listener", l.Name)
		log.Info("Processing listener", "spec", l)

		listenerObj, err := r.buildListener(l)
		if err != nil {
			return cache.Snapshot{}, fmt.Errorf("failed to build listener %s: %w", l.Name, err)
		}

		listeners = append(listeners, listenerObj)
	}

	version := strconv.FormatInt(time.Now().Unix(), 10)
	snapshot, err := cache.NewSnapshot(version,
		map[res.Type][]types.Resource{
			res.EndpointType: endpoints,
			res.ClusterType:  clusters,
			res.ListenerType: listeners,
		},
	)

	if err != nil {
		log.Error(err, "failed to create xDS snapshot")
		return cache.Snapshot{}, err
	}

	log.Info("xDS snapshot created", "version", version)
	return *snapshot, nil
}

func (r *XDSControlPlaneReconciler) buildCluster(ctx context.Context, c api.ClusterSpec) (*cluster.Cluster, *endpoint.ClusterLoadAssignment, error) {
	log := ctrlLog.FromContext(ctx).WithValues("cluster", c.Name)

	// Build endpoints if specified
	var cla *endpoint.ClusterLoadAssignment
	if c.LoadAssignment != nil && c.LoadAssignment.EndpointsFrom != nil {
		addrs, err := r.discoverEndpoints(ctx, c.LoadAssignment.EndpointsFrom)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to discover endpoints: %w", err)
		}

		log.Info("Discovered node addresses", "addresses", addrs)

		lbs := make([]*endpoint.LbEndpoint, 0, len(addrs))
		port := int32(c.LoadAssignment.EndpointsFrom.Port)

		for _, ip := range addrs {
			lbs = append(lbs, &endpoint.LbEndpoint{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Address:       ip,
									PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(port)},
								},
							},
						},
					},
				},
			})
		}

		cla = &endpoint.ClusterLoadAssignment{
			ClusterName: c.Name,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: lbs,
			}},
		}
	}

	// Build cluster
	clusterObj := &cluster.Cluster{
		Name:                 c.Name,
		ConnectTimeout:       durationpb.New(time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
	}

	if cla != nil {
		clusterObj.LoadAssignment = cla
	}

	// Set cluster type
	switch c.Type {
	case "static":
		clusterObj.ClusterDiscoveryType = &cluster.Cluster_Type{Type: cluster.Cluster_STATIC}
	case "strict_dns":
		clusterObj.ClusterDiscoveryType = &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS}
	}

	// Set load balancing policy
	switch c.LbPolicy {
	case "round_robin":
		clusterObj.LbPolicy = cluster.Cluster_ROUND_ROBIN
	case "least_request":
		clusterObj.LbPolicy = cluster.Cluster_LEAST_REQUEST
	}

	// Handle transport socket
	if c.TransportSocket != nil {
		anyTS, err := r.jsonToAny(c.TransportSocket.Name, c.TransportSocket.TypedConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert transport socket config: %w", err)
		}
		clusterObj.TransportSocket = &core.TransportSocket{
			Name: c.TransportSocket.Name,
			ConfigType: &core.TransportSocket_TypedConfig{
				TypedConfig: anyTS,
			},
		}
	}

	// Handle health check configuration
	if c.HealthCheck != nil {
		healthCheck, err := r.buildHealthCheck(c.HealthCheck)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build health check config: %w", err)
		}
		clusterObj.HealthChecks = []*core.HealthCheck{healthCheck}
		log.Info("Added health check to cluster", "healthCheck", healthCheck)
	}

	return clusterObj, cla, nil
}

func (r *XDSControlPlaneReconciler) buildHealthCheck(hc *api.HealthCheckSpec) (*core.HealthCheck, error) {
	healthCheck := &core.HealthCheck{}

	// Set timeout
	if hc.Timeout != "" {
		timeout, err := time.ParseDuration(hc.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid health check timeout: %w", err)
		}
		healthCheck.Timeout = durationpb.New(timeout)
	} else {
		healthCheck.Timeout = durationpb.New(5 * time.Second) // Default timeout
	}

	// Set interval
	if hc.Interval != "" {
		interval, err := time.ParseDuration(hc.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid health check interval: %w", err)
		}
		healthCheck.Interval = durationpb.New(interval)
	} else {
		healthCheck.Interval = durationpb.New(10 * time.Second) // Default interval
	}

	// Set interval jitter
	if hc.IntervalJitter != "" {
		jitter, err := time.ParseDuration(hc.IntervalJitter)
		if err != nil {
			return nil, fmt.Errorf("invalid health check interval jitter: %w", err)
		}
		healthCheck.IntervalJitter = durationpb.New(jitter)
	}

	// Set thresholds
	if hc.UnhealthyThreshold > 0 {
		healthCheck.UnhealthyThreshold = wrapperspb.UInt32(uint32(hc.UnhealthyThreshold))
	} else {
		healthCheck.UnhealthyThreshold = wrapperspb.UInt32(3)
	}

	if hc.HealthyThreshold > 0 {
		healthCheck.HealthyThreshold = wrapperspb.UInt32(uint32(hc.HealthyThreshold))
	} else {
		healthCheck.HealthyThreshold = wrapperspb.UInt32(2)
	}

	// Set reuse connection
	healthCheck.ReuseConnection = wrapperspb.Bool(hc.ReuseConnection)

	// Configure specific health check type
	if hc.HTTPHealthCheck != nil {
		httpHC, err := r.buildHTTPHealthCheck(hc.HTTPHealthCheck)
		if err != nil {
			return nil, fmt.Errorf("failed to build HTTP health check: %w", err)
		}
		healthCheck.HealthChecker = &core.HealthCheck_HttpHealthCheck_{
			HttpHealthCheck: httpHC,
		}
	} else if hc.TCPHealthCheck != nil {
		tcpHC := r.buildTCPHealthCheck(hc.TCPHealthCheck)
		healthCheck.HealthChecker = &core.HealthCheck_TcpHealthCheck_{
			TcpHealthCheck: tcpHC,
		}
	} else if hc.GRPCHealthCheck != nil {
		grpcHC := r.buildGRPCHealthCheck(hc.GRPCHealthCheck)
		healthCheck.HealthChecker = &core.HealthCheck_GrpcHealthCheck_{
			GrpcHealthCheck: grpcHC,
		}
	} else {
		// Default to TCP health check if no specific type is configured
		tcpHC := &core.HealthCheck_TcpHealthCheck{}
		healthCheck.HealthChecker = &core.HealthCheck_TcpHealthCheck_{
			TcpHealthCheck: tcpHC,
		}
	}

	return healthCheck, nil
}

func (r *XDSControlPlaneReconciler) buildHTTPHealthCheck(hc *api.HTTPHealthCheckSpec) (*core.HealthCheck_HttpHealthCheck, error) {
	httpHC := &core.HealthCheck_HttpHealthCheck{
		Path: hc.Path,
	}

	if hc.Host != "" {
		httpHC.Host = hc.Host
	}

	// Add request headers
	for _, header := range hc.RequestHeadersToAdd {
		httpHC.RequestHeadersToAdd = append(httpHC.RequestHeadersToAdd, &core.HeaderValueOption{
			Header: &core.HeaderValue{
				Key:   header.Header.Key,
				Value: header.Header.Value,
			},
			Append: wrapperspb.Bool(header.Append),
		})
	}

	// Add expected status codes
	for _, statusRange := range hc.ExpectedStatuses {
		httpHC.ExpectedStatuses = append(httpHC.ExpectedStatuses, &envoytype.Int64Range{
			Start: statusRange.Start,
			End:   statusRange.End,
		})
	}

	// Default expected status if none specified
	if len(httpHC.ExpectedStatuses) == 0 {
		httpHC.ExpectedStatuses = []*envoytype.Int64Range{
			{Start: 200, End: 299},
		}
	}

	return httpHC, nil
}

func (r *XDSControlPlaneReconciler) buildTCPHealthCheck(hc *api.TCPHealthCheckSpec) *core.HealthCheck_TcpHealthCheck {
	tcpHC := &core.HealthCheck_TcpHealthCheck{}

	if len(hc.Send) > 0 {
		tcpHC.Send = &core.HealthCheck_Payload{
			Payload: &core.HealthCheck_Payload_Binary{
				Binary: hc.Send,
			},
		}
	}

	for _, receive := range hc.Receive {
		tcpHC.Receive = append(tcpHC.Receive, &core.HealthCheck_Payload{
			Payload: &core.HealthCheck_Payload_Binary{
				Binary: receive,
			},
		})
	}

	return tcpHC
}

func (r *XDSControlPlaneReconciler) buildGRPCHealthCheck(hc *api.GRPCHealthCheckSpec) *core.HealthCheck_GrpcHealthCheck {
	grpcHC := &core.HealthCheck_GrpcHealthCheck{}

	if hc.ServiceName != "" {
		grpcHC.ServiceName = hc.ServiceName
	}

	if hc.Authority != "" {
		grpcHC.Authority = hc.Authority
	}

	return grpcHC
}

func (r *XDSControlPlaneReconciler) buildListener(l api.ListenerSpec) (*listener.Listener, error) {
	lf := make([]*listener.ListenerFilter, 0, len(l.ListenerFilters))
	for _, f := range l.ListenerFilters {
		listenerFilter := &listener.ListenerFilter{Name: f.Name}

		// Add typed config if provided
		if len(f.TypedConfig.Raw) > 0 {
			anyCfg, err := r.jsonToAny(f.Name, f.TypedConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to convert listener filter config: %w", err)
			}
			listenerFilter.ConfigType = &listener.ListenerFilter_TypedConfig{
				TypedConfig: anyCfg,
			}
		}

		lf = append(lf, listenerFilter)
	}

	fc := make([]*listener.FilterChain, 0, len(l.FilterChains))
	for _, chain := range l.FilterChains {
		filters := make([]*listener.Filter, 0, len(chain.Filters))
		for _, f := range chain.Filters {
			anyCfg, err := r.jsonToAny(f.Name, f.TypedConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to convert filter config: %w", err)
			}
			filters = append(filters, &listener.Filter{
				Name: f.Name,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: anyCfg,
				},
			})
		}
		fc = append(fc, &listener.FilterChain{Filters: filters})
	}

	// Process access logs
	var accessLogs []*accesslog.AccessLog
	for _, al := range l.AccessLog {
		anyConfig, err := r.jsonToAny(al.Name, al.TypedConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to convert access log config: %w", err)
		}
		accessLogs = append(accessLogs, &accesslog.AccessLog{
			Name: al.Name,
			ConfigType: &accesslog.AccessLog_TypedConfig{
				TypedConfig: anyConfig,
			},
		})
	}
	return &listener.Listener{
		Name: l.Name,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address:       l.Address,
					PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(l.Port)},
				},
			},
		},
		ListenerFilters: lf,
		FilterChains:    fc,
		AccessLog:       accessLogs,
	}, nil
}

func (r *XDSControlPlaneReconciler) discoverEndpoints(ctx context.Context, selector *api.EndpointSelectorSpec) ([]string, error) {
	var addrs []string

	if selector.Type == "Node" && selector.Selector != nil {
		labelSelector, err := metav1.LabelSelectorAsSelector(selector.Selector)
		if err != nil {
			return nil, fmt.Errorf("invalid label selector: %w", err)
		}

		var nodeList corev1.NodeList
		if err := r.List(ctx, &nodeList, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
			return nil, fmt.Errorf("failed to list nodes: %w", err)
		}

		for _, node := range nodeList.Items {
			for _, addr := range node.Status.Addresses {
				if addr.Type == corev1.NodeInternalIP {
					addrs = append(addrs, addr.Address)
					break
				}
			}
		}
	}

	return addrs, nil
}

func (r *XDSControlPlaneReconciler) jsonToAny(typeURL string, in apiextensionsv1.JSON) (*anypb.Any, error) {
	log := ctrlLog.Log.WithValues("typeURL", typeURL)
	log.Info("Converting JSON to protobuf Any with proper protobuf marshaling")

	if len(in.Raw) == 0 {
		return nil, fmt.Errorf("empty JSON config for type %s", typeURL)
	}

	// Parse JSON to extract @type and remove it from the config
	var m map[string]interface{}
	if err := json.Unmarshal(in.Raw, &m); err != nil {
		log.Error(err, "failed to unmarshal JSON")
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Verify @type matches the expected typeURL
	if jsonType, ok := m["@type"].(string); ok {
		if jsonType != typeURL {
			log.Info("typeURL mismatch", "expected", typeURL, "found", jsonType)
			// Use the @type from JSON if it exists
			typeURL = jsonType
		}
		// Remove @type from the config before protobuf unmarshaling
		delete(m, "@type")
	}

	// Marshal back to JSON without @type
	filteredJSON, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filtered JSON: %w", err)
	}

	log.Info("Converting config", "typeURL", typeURL, "config", string(filteredJSON))

	// Convert to appropriate protobuf message based on typeURL
	switch typeURL {
	case "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy":
		var msg tcp_proxy.TcpProxy
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal tcp_proxy config")
			return nil, fmt.Errorf("failed to unmarshal tcp_proxy config: %w", err)
		}
		return anypb.New(&msg)

	case "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager":
		var msg http_connection_manager.HttpConnectionManager
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal http_connection_manager config")
			return nil, fmt.Errorf("failed to unmarshal http_connection_manager config: %w", err)
		}
		return anypb.New(&msg)

	case "type.googleapis.com/envoy.extensions.transport_sockets.proxy_protocol.v3.ProxyProtocolUpstreamTransport":
		var msg proxy_protocol.ProxyProtocolUpstreamTransport
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal proxy_protocol upstream transport config")
			return nil, fmt.Errorf("failed to unmarshal proxy_protocol upstream transport config: %w", err)
		}
		return anypb.New(&msg)

	case "type.googleapis.com/envoy.extensions.transport_sockets.proxy_protocol.v3.ProxyProtocolConfig":
		var msg core.ProxyProtocolConfig
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal proxy_protocol config")
			return nil, fmt.Errorf("failed to unmarshal proxy_protocol config: %w", err)
		}
		return anypb.New(&msg)

	case "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext":
		var msg tls.DownstreamTlsContext
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal downstream_tls_context config")
			return nil, fmt.Errorf("failed to unmarshal downstream_tls_context config: %w", err)
		}
		return anypb.New(&msg)

	case "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext":
		var msg tls.UpstreamTlsContext
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal upstream_tls_context config")
			return nil, fmt.Errorf("failed to unmarshal upstream_tls_context config: %w", err)
		}
		return anypb.New(&msg)

	case "type.googleapis.com/envoy.extensions.transport_sockets.raw_buffer.v3.RawBuffer":
		var msg raw_buffer.RawBuffer
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal raw_buffer transport config")
			return nil, fmt.Errorf("failed to unmarshal raw_buffer transport config: %w", err)
		}
		return anypb.New(&msg)

	case "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog":
		var msg file_access_log.FileAccessLog
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal file_access_log config")
			return nil, fmt.Errorf("failed to unmarshal file_access_log config: %w", err)
		}
		return anypb.New(&msg)

	case "type.googleapis.com/envoy.extensions.filters.listener.proxy_protocol.v3.ProxyProtocol":
		var msg listener_proxy_protocol.ProxyProtocol
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal listener_proxy_protocol config")
			return nil, fmt.Errorf("failed to unmarshal listener_proxy_protocol config: %w", err)
		}
		return anypb.New(&msg)
	default:
		log.Info("Unknown typeURL, trying generic protobuf conversion", "typeURL", typeURL)
		// For unknown types, create Any with the JSON data as value
		// This is a fallback that may work for some types
		any := &anypb.Any{
			TypeUrl: typeURL,
			Value:   filteredJSON,
		}
		log.Info("Created generic Any message", "typeURL", typeURL)
		return any, nil
	}
}
