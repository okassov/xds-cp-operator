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

type ClusterSpec struct {
	Name            string               `json:"name"`
	Type            string               `json:"type"`
	LbPolicy        string               `json:"lbPolicy"`
	ConnectTimeout  string               `json:"connectTimeout,omitempty"`
	TransportSocket *TransportSocketSpec `json:"transportSocket,omitempty"`
	LoadAssignment  *LoadAssignmentSpec  `json:"loadAssignment,omitempty"`
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
	Name            string            `json:"name"`
	Address         string            `json:"address"`
	Port            int               `json:"port"`
	ListenerFilters []string          `json:"listenerFilters,omitempty"`
	FilterChains    []FilterChainSpec `json:"filterChains"`
	// +kubebuilder:validation:Optional
	AccessLog []AccessLogSpec `json:"accessLog,omitempty"`
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
