/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of Tenant. Edit tenant_types.go to remove/update
	// +optional
	Foo           *string            `json:"foo,omitempty"`
	Owner         string             `json:"owner"`
	Namespace     NamespaceSpec      `json:"namespace"`
	Quota         *QuotaSpec         `json:"quota,omitempty"`
	LimitRange    *LimitRangeSpec    `json:"limitRange,omitempty"`
	NetworkPolicy *NetworkPolicySpec `json:"networkPolicy,omitempty"`
	RBAC          *RBACSpec          `json:"rbac,omitempty"`
}
type NamespaceSpec struct {
	Name string `json:"name"`
}

type QuotaSpec struct {
	CPU                    string `json:"cpu,omitempty"`
	Memory                 string `json:"memory,omitempty"`
	Pods                   int32  `json:"pods,omitempty"`
	Services               int32  `json:"services,omitempty"`
	PersistentVolumeClaims int32  `json:"persistentVolumeClaims,omitempty"`
}

type LimitRangeSpec struct {
	DefaultCPURequest    string `json:"defaultCpuRequest,omitempty"`
	DefaultMemoryRequest string `json:"defaultMemoryRequest,omitempty"`
	DefaultCPULimit      string `json:"defaultCpuLimit,omitempty"`
	DefaultMemoryLimit   string `json:"defaultMemoryLimit,omitempty"`
}

type NetworkPolicyMode string

const (
	NetworkPolicyModeIsolated NetworkPolicyMode = "Isolated"
	NetworkPolicyModeOpen     NetworkPolicyMode = "Open"
)

type NetworkPolicySpec struct {
	Mode NetworkPolicyMode `json:"mode,omitempty"`
}

type RBACSpec struct {
	AdminUsers []string `json:"adminUsers,omitempty"`
	ViewUsers  []string `json:"viewUsers,omitempty"`
}

type TenantPhase string

const (
	TenantPhasePending  TenantPhase = "Pending"
	TenantPhaseCreating TenantPhase = "Creating"
	TenantPhaseReady    TenantPhase = "Ready"
	TenantPhaseUpdating TenantPhase = "Updating"
	TenantPhaseDeleting TenantPhase = "Deleting"
	TenantPhaseFailed   TenantPhase = "Failed"
)

// TenantStatus defines the observed state of Tenant.
type TenantStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Tenant resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	Phase              TenantPhase        `json:"phase,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Namespace          string             `json:"namespace,omitempty"`
	LastError          string             `json:"lastError,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Tenant is the Schema for the tenants API
type Tenant struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Tenant
	// +required
	Spec TenantSpec `json:"spec"`

	// status defines the observed state of Tenant
	// +optional
	Status TenantStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TenantList contains a list of Tenant
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}
