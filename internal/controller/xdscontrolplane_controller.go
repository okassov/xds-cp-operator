package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
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

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	"sync"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type XDSControlPlaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type XDSServerSingleton struct {
	sync.Mutex
	Started bool
}

var xdsServer XDSServerSingleton
var sharedSnapCache cache.SnapshotCache

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
		log.Error(err, "unable to fetch XDSControlPlane")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	xdsServer.Lock()
	if !xdsServer.Started {
		log.Info("Starting xDS gRPC server", "port", xdsCRD.Spec.XdsPort)
		go startXDSServer(ctx, &xdsCRD)
		xdsServer.Started = true
	}
	xdsServer.Unlock()

	if sharedSnapCache == nil {
		log.Info("xDS server not started yet; cannot set snapshot")
		return ctrl.Result{Requeue: true}, nil
	}

	snapshot, err := buildXDSSnapshot(ctx, r.Client, &xdsCRD)
	if err != nil {
		log.Error(err, "Error building xDS snapshot")
		return ctrl.Result{}, err
	}

	nodeID := "external-envoy"
	log.Info("Setting xDS snapshot", "nodeID", nodeID)
	if err := sharedSnapCache.SetSnapshot(ctx, nodeID, &snapshot); err != nil {
		log.Error(err, "failed to set xDS snapshot")
		return ctrl.Result{}, err
	}
	log.Info("Successfully set xDS snapshot", "nodeID", nodeID)
	return ctrl.Result{}, nil
}

func startXDSServer(ctx context.Context, crd *api.XDSControlPlane) {
	log := ctrlLog.FromContext(ctx).WithValues("xdscontrolplane", crd.Name)
	log.Info("Starting xDS gRPC server entry", "crd", crd.Name)
	defer log.Info("Exiting xDS gRPC server", "crd", crd.Name)
	port := crd.Spec.XdsPort
	if port == 0 {
		port = 18000 // default xDS port
		log.Info("No xdsPort specified, using default", "port", port)
	}
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error(err, "failed to listen for xDS server", "addr", addr)
		return
	}
	log.Info("xDS gRPC server listening", "addr", addr)

	if sharedSnapCache == nil {
		log.Info("Initializing shared xDS snapshot cache")
		sharedSnapCache = cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	}
	srv := grpc.NewServer()
	xdsServer := serverv3.NewServer(ctx, sharedSnapCache, nil)

	log.Info("Registering xDS gRPC services")
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(srv, xdsServer)
	endpointgrpc.RegisterEndpointDiscoveryServiceServer(srv, xdsServer)
	clustergrpc.RegisterClusterDiscoveryServiceServer(srv, xdsServer)
	listenergrpc.RegisterListenerDiscoveryServiceServer(srv, xdsServer)
	routegrpc.RegisterRouteDiscoveryServiceServer(srv, xdsServer)

	go func() {
		log.Info("xDS gRPC server serving")
		if err := srv.Serve(lis); err != nil {
			log.Error(err, "xDS gRPC server exited with error")
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down xDS gRPC server")
	srv.GracefulStop()
}

func buildXDSSnapshot(ctx context.Context, k8s client.Client, crd *api.XDSControlPlane) (cache.Snapshot, error) {
	log := ctrlLog.FromContext(ctx).WithValues("xdscontrolplane", crd.Name)
	log.Info("Building xDS snapshot", "crd", crd.Name)
	defer log.Info("Finished building xDS snapshot", "crd", crd.Name)
	var endpoints []types.Resource
	var clusters []types.Resource
	var listeners []types.Resource

	// --- Endpoints (EDS) + Clusters (CDS)
	for _, c := range crd.Spec.Clusters {
		log := log.WithValues("cluster", c.Name)
		log.Info("Processing cluster", "spec", c)
		var addrs []string
		if c.LoadAssignment != nil && c.LoadAssignment.EndpointsFrom != nil {
			selector := c.LoadAssignment.EndpointsFrom.Selector
			if c.LoadAssignment.EndpointsFrom.Type == "Node" && selector != nil {
				labelSelector, _ := metav1.LabelSelectorAsSelector(selector)
				var nodeList corev1.NodeList
				if err := k8s.List(ctx, &nodeList, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
					log.Error(err, "failed to get nodes")
				}
				for _, node := range nodeList.Items {
					for _, addr := range node.Status.Addresses {
						if addr.Type == corev1.NodeInternalIP {
							addrs = append(addrs, addr.Address)
							break
						}
					}
				}
				log.Info("Discovered node addresses", "addresses", addrs)
			}
		}
		lbs := []*endpoint.LbEndpoint{}
		port := int32(0)
		if c.LoadAssignment != nil && c.LoadAssignment.EndpointsFrom != nil {
			port = int32(c.LoadAssignment.EndpointsFrom.Port)
		}
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
		cla := &endpoint.ClusterLoadAssignment{
			ClusterName: c.Name,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: lbs,
			}},
		}
		endpoints = append(endpoints, cla)

		// Build Cluster
		clusterObj := &cluster.Cluster{
			Name:                 c.Name,
			ConnectTimeout:       durationpb.New(1_000_000_000), // default 1s
			ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
			LbPolicy:             cluster.Cluster_ROUND_ROBIN,
			LoadAssignment:       cla,
		}
		if c.Type != "" {
			switch c.Type {
			case "static":
				clusterObj.ClusterDiscoveryType = &cluster.Cluster_Type{Type: cluster.Cluster_STATIC}
			case "strict_dns":
				clusterObj.ClusterDiscoveryType = &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS}
			}
		}
		if c.LbPolicy != "" {
			switch c.LbPolicy {
			case "round_robin":
				clusterObj.LbPolicy = cluster.Cluster_ROUND_ROBIN
			case "least_request":
				clusterObj.LbPolicy = cluster.Cluster_LEAST_REQUEST
			}
		}
		if c.ConnectTimeout != "" {
			// TODO: parse user value
		}
		if c.TransportSocket != nil {
			anyTS, _ := jsonToAny("", c.TransportSocket.TypedConfig)
			clusterObj.TransportSocket = &core.TransportSocket{
				Name: c.TransportSocket.Name,
				ConfigType: &core.TransportSocket_TypedConfig{
					TypedConfig: anyTS,
				},
			}
		}
		clusters = append(clusters, clusterObj)
	}

	// --- Listeners (LDS)
	for _, l := range crd.Spec.Listeners {
		log := log.WithValues("listener", l.Name)
		log.Info("Processing listener", "spec", l)
		lf := []*listener.ListenerFilter{}
		for _, f := range l.ListenerFilters {
			lf = append(lf, &listener.ListenerFilter{Name: f})
		}
		fc := []*listener.FilterChain{}
		for _, chain := range l.FilterChains {
			filters := []*listener.Filter{}
			for _, f := range chain.Filters {
				anyCfg, _ := jsonToAny("", f.TypedConfig)
				filters = append(filters, &listener.Filter{
					Name: f.Name,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: anyCfg,
					},
				})
			}
			fc = append(fc, &listener.FilterChain{Filters: filters})
		}
		listenerObj := &listener.Listener{
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
		}
		listeners = append(listeners, listenerObj)
	}

	snapshot, err := cache.NewSnapshot("1",
		map[res.Type][]types.Resource{
			res.EndpointType: endpoints,
			res.ClusterType:  clusters,
			res.ListenerType: listeners,
		},
	)
	if err != nil {
		log.Error(err, "failed to create xDS snapshot")
	} else {
		log.Info("xDS snapshot created", "version", "1")
	}
	return *snapshot, err
}

func jsonToAny(_ string, in apiextensionsv1.JSON) (*anypb.Any, error) {
	log := ctrlLog.Log.WithValues("typedConfig", string(in.Raw))
	log.Info("Converting JSON to protobuf Any (with binary protobuf)")

	// Парсим JSON чтобы достать "@type" и удалить его из JSON перед protojson.Unmarshal
	var m map[string]interface{}
	if err := json.Unmarshal(in.Raw, &m); err != nil {
		log.Error(err, "unmarshal failed")
		return nil, err
	}
	typeURL, ok := m["@type"].(string)
	if !ok || typeURL == "" {
		log.Error(errors.New("missing @type"), "typedConfig must contain @type with full type URL")
		return nil, errors.New("typedConfig missing @type")
	}
	delete(m, "@type")

	// Marshal обратно в JSON без "@type"
	filteredJSON, err := json.Marshal(m)
	if err != nil {
		log.Error(err, "failed to marshal filtered JSON without @type")
		return nil, err
	}

	switch typeURL {
	case "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy":
		var msg tcp_proxy.TcpProxy
		if err := protojson.Unmarshal(filteredJSON, &msg); err != nil {
			log.Error(err, "failed to unmarshal tcp_proxy config")
			return nil, err
		}
		bin, err := proto.Marshal(&msg)
		if err != nil {
			log.Error(err, "failed to marshal tcp_proxy to binary proto")
			return nil, err
		}
		return &anypb.Any{
			TypeUrl: typeURL,
			Value:   bin,
		}, nil
	default:
		log.Error(errors.New("unsupported typeURL for binary conversion"), "unknown typedConfig type, returning as raw json")
		return &anypb.Any{
			TypeUrl: typeURL,
			Value:   filteredJSON,
		}, nil
	}
}
