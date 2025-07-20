package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EndpointSelectorSpec struct {
	Type      string                `json:"type"`
	Selector  *metav1.LabelSelector `json:"selector,omitempty"`
	Port      int                   `json:"port,omitempty"`
	Name      string                `json:"name,omitempty"`
	Namespace string                `json:"namespace,omitempty"`
}

type LoadAssignmentSpec struct {
	EndpointsFrom *EndpointSelectorSpec `json:"endpointsFrom,omitempty"`
}

type TransportSocketSpec struct {
	Name        string               `json:"name"`
	TypedConfig apiextensionsv1.JSON `json:"typedConfig,omitempty"`
}

// HealthCheckSpec defines health check configuration for a cluster
type HealthCheckSpec struct {
	// +kubebuilder:validation:Optional
	// Timeout specifies the time to wait for a health check response
	Timeout string `json:"timeout,omitempty"`

	// +kubebuilder:validation:Optional
	// Interval specifies the interval between health checks
	Interval string `json:"interval,omitempty"`

	// +kubebuilder:validation:Optional
	// IntervalJitter specifies the amount of jitter to add to the interval
	IntervalJitter string `json:"intervalJitter,omitempty"`

	// +kubebuilder:validation:Optional
	// UnhealthyThreshold specifies the number of unhealthy health checks before marking the host as unhealthy
	UnhealthyThreshold int32 `json:"unhealthyThreshold,omitempty"`

	// +kubebuilder:validation:Optional
	// HealthyThreshold specifies the number of healthy health checks before marking the host as healthy
	HealthyThreshold int32 `json:"healthyThreshold,omitempty"`

	// +kubebuilder:validation:Optional
	// ReuseConnection specifies whether to reuse health check connections
	ReuseConnection bool `json:"reuseConnection,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTPHealthCheck specifies HTTP health check configuration
	HTTPHealthCheck *HTTPHealthCheckSpec `json:"httpHealthCheck,omitempty"`

	// +kubebuilder:validation:Optional
	// TCPHealthCheck specifies TCP health check configuration
	TCPHealthCheck *TCPHealthCheckSpec `json:"tcpHealthCheck,omitempty"`

	// +kubebuilder:validation:Optional
	// GRPCHealthCheck specifies gRPC health check configuration
	GRPCHealthCheck *GRPCHealthCheckSpec `json:"grpcHealthCheck,omitempty"`
}

// HTTPHealthCheckSpec defines HTTP health check configuration
type HTTPHealthCheckSpec struct {
	// +kubebuilder:validation:Required
	// Path specifies the HTTP path for health checks
	Path string `json:"path"`

	// +kubebuilder:validation:Optional
	// Host specifies the value of the host header in the HTTP health check request
	Host string `json:"host,omitempty"`

	// +kubebuilder:validation:Optional
	// RequestHeadersToAdd specifies headers to add to health check requests
	RequestHeadersToAdd []HeaderValueOptionSpec `json:"requestHeadersToAdd,omitempty"`

	// +kubebuilder:validation:Optional
	// ExpectedStatuses specifies the expected HTTP status codes for a successful health check
	ExpectedStatuses []HTTPStatusRangeSpec `json:"expectedStatuses,omitempty"`
}

// TCPHealthCheckSpec defines TCP health check configuration
type TCPHealthCheckSpec struct {
	// +kubebuilder:validation:Optional
	// Send specifies the bytes to send during TCP health check
	Send []byte `json:"send,omitempty"`

	// +kubebuilder:validation:Optional
	// Receive specifies the bytes expected in response during TCP health check
	Receive [][]byte `json:"receive,omitempty"`
}

// GRPCHealthCheckSpec defines gRPC health check configuration
type GRPCHealthCheckSpec struct {
	// +kubebuilder:validation:Optional
	// ServiceName specifies the service name to use in gRPC health checks
	ServiceName string `json:"serviceName,omitempty"`

	// +kubebuilder:validation:Optional
	// Authority specifies the :authority header value to use in gRPC health checks
	Authority string `json:"authority,omitempty"`
}

// HeaderValueOptionSpec defines header value configuration
type HeaderValueOptionSpec struct {
	// +kubebuilder:validation:Required
	Header HeaderValueSpec `json:"header"`

	// +kubebuilder:validation:Optional
	Append bool `json:"append,omitempty"`
}

// HeaderValueSpec defines header name and value
type HeaderValueSpec struct {
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// HTTPStatusRangeSpec defines HTTP status code range
type HTTPStatusRangeSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=599
	Start int64 `json:"start"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=599
	End int64 `json:"end"`
}

type ClusterSpec struct {
	Name            string               `json:"name"`
	Type            string               `json:"type"`
	LbPolicy        string               `json:"lbPolicy"`
	ConnectTimeout  string               `json:"connectTimeout,omitempty"`
	TransportSocket *TransportSocketSpec `json:"transportSocket,omitempty"`
	LoadAssignment  *LoadAssignmentSpec  `json:"loadAssignment,omitempty"`
	// +kubebuilder:validation:Optional
	HealthCheck *HealthCheckSpec `json:"healthCheck,omitempty"`
}

// FilterSpec defines the Envoy filter configuration
type FilterSpec struct {
	Name        string               `json:"name"`
	TypedConfig apiextensionsv1.JSON `json:"typedConfig"`
}

type FilterChainSpec struct {
	Filters []FilterSpec `json:"filters"`
}

// ListenerSpec defines the Envoy listener configuration
type ListenerSpec struct {
	Name            string               `json:"name"`
	Address         string               `json:"address"`
	Port            int                  `json:"port"`
	ListenerFilters []ListenerFilterSpec `json:"listenerFilters,omitempty"`
	FilterChains    []FilterChainSpec    `json:"filterChains"`
	// +kubebuilder:validation:Optional
	AccessLog []AccessLogSpec `json:"accessLog,omitempty"`
}

// ListenerFilterSpec defines the listener filter configuration
type ListenerFilterSpec struct {
	Name        string               `json:"name"`
	TypedConfig apiextensionsv1.JSON `json:"typedConfig,omitempty"`
}

// AccessLogSpec defines access log configuration
type AccessLogSpec struct {
	Name        string               `json:"name"`
	TypedConfig apiextensionsv1.JSON `json:"typedConfig"`
}

type VirtualHostSpec struct {
	Name    string                 `json:"name"`
	Domains []string               `json:"domains"`
	Routes  []apiextensionsv1.JSON `json:"routes"`
}

type RouteConfigSpec struct {
	Name         string            `json:"name"`
	VirtualHosts []VirtualHostSpec `json:"virtualHosts"`
}

type XDSControlPlaneSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	XdsPort int `json:"xdsPort"`

	// +kubebuilder:validation:Optional
	// NodeIDs specifies the list of Envoy node IDs that should receive this configuration
	// If empty, defaults to ["external-envoy"]
	NodeIDs []string `json:"nodeIDs,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Listeners []ListenerSpec `json:"listeners"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Clusters []ClusterSpec `json:"clusters"`

	// +kubebuilder:validation:Optional
	Routes []RouteConfigSpec `json:"routes,omitempty"`
}

// XDSControlPlaneStatus defines the observed state of XDSControlPlane
type XDSControlPlaneStatus struct {
	// Phase represents the current phase of the XDSControlPlane
	// +kubebuilder:validation:Enum=Pending;Ready;Error
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the XDSControlPlane's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ConnectedNodeIDs lists the Envoy node IDs currently connected to the xDS server
	// +optional
	ConnectedNodeIDs []string `json:"connectedNodeIDs,omitempty"`

	// XdsServerAddress is the address where the xDS server is listening
	// +optional
	XdsServerAddress string `json:"xdsServerAddress,omitempty"`

	// LastSnapshotVersion indicates the version of the last successfully created snapshot
	// +optional
	LastSnapshotVersion string `json:"lastSnapshotVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="XDS Port",type=integer,JSONPath=`.spec.xdsPort`
// +kubebuilder:printcolumn:name="Connected Nodes",type=string,JSONPath=`.status.connectedNodeIDs`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type XDSControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   XDSControlPlaneSpec   `json:"spec,omitempty"`
	Status XDSControlPlaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type XDSControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []XDSControlPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&XDSControlPlane{}, &XDSControlPlaneList{})
}
