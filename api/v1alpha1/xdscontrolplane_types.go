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

type FilterSpec struct {
	Name        string               `json:"name"`
	TypedConfig apiextensionsv1.JSON `json:"typedConfig,omitempty"`
}

type FilterChainSpec struct {
	Filters []FilterSpec `json:"filters"`
}

type ListenerSpec struct {
	Name            string            `json:"name"`
	Address         string            `json:"address"`
	Port            int               `json:"port"`
	ListenerFilters []string          `json:"listenerFilters,omitempty"`
	FilterChains    []FilterChainSpec `json:"filterChains"`
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
	XdsPort   int               `json:"xdsPort"`
	Listeners []ListenerSpec    `json:"listeners"`
	Clusters  []ClusterSpec     `json:"clusters"`
	Routes    []RouteConfigSpec `json:"routes,omitempty"`
}

// +kubebuilder:object:root=true
type XDSControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec XDSControlPlaneSpec `json:"spec,omitempty"`
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
