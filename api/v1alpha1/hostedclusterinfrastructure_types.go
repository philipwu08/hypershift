package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CurrentPhase string
type fieldStatus string

const (
	PhaseInit   CurrentPhase = "INIT"
	PhaseInfra  CurrentPhase = "INFRA"
	PhaseIAM    CurrentPhase = "IAM"
	PhaseReady  CurrentPhase = "READY"
	PhaseVerify CurrentPhase = "VERIFYING" // or validating??

	fieldVerified      fieldStatus = "Verified"
	fieldAsExpected    fieldStatus = "AsExpected"
	fieldNotAsExpected fieldStatus = "NotAsExpected"
)

func init() {
	SchemeBuilder.Register(&HostedClusterInfrastructure{}, &HostedClusterInfrastructureList{})
}

// HostedClusterSpec is the desired behavior of a HostedCluster.
type HostedClusterInfrastructureSpec struct {

	// AltInfraID is a globally unique identifier for the cluster. This identifier
	// will be used to associate various cloud resources with the HostedCluster
	// and its associated NodePools. If not specified the metadata.name for this
	// resource is used. When specified, this value is used.
	//
	// +optional
	// +immutable
	AltInfraID string `json:"altInfraID,omitempty"`

	// Platform specifies the underlying infrastructure provider for the cluster
	// and is used to configure platform specific behavior.
	//
	// +immutable
	Platform PlatformInfraSpec `json:"platform"`

	// DNS specifies DNS configuration for the cluster.
	//
	// +immutable
	DNS DNSSpec `json:"dns"`

	// Networking specifies network configuration for the cluster.
	//
	// +optional
	// +immutable
	Networking ClusterNetworkingInfraSpec `json:"networking,omitempty"`

	// CloudProvider secret, contains the Cloud credenetial and Base Domain
	// When not present, we expect all values to populated at create time
	// This can be from the hypershift cli or via a kubectl create.
	// +optional
	CloudProvider corev1.LocalObjectReference `json:"cloudProvider,omitempty"`
}

// HostedClusterSpec is the desired behavior of a HostedCluster.
type AWSHostedClusterInfrastructureStatus struct {
}

type ClusterNetworkingInfraSpec struct {
	// MachineNetwork is the list of IP address pools for machines.
	// TODO: make this required in the next version of the API
	//
	// +immutable
	// +optional
	MachineNetwork []MachineNetworkEntry `json:"machineNetwork,omitempty"`
}

// PlatformSpec specifies the underlying infrastructure provider for the cluster
// and is used to configure platform specific behavior.
type PlatformInfraSpec struct {
	// Type is the type of infrastructure provider for the cluster.
	//
	// +kubebuilder:validation:Enum=AWS;Azure;PowerVS
	// +immutable
	Type PlatformType `json:"type"`

	// AWS specifies configuration for clusters running on Amazon Web Services.
	//
	// +optional
	// +immutable
	AWS *AWSPlatformInfraSpec `json:"aws,omitempty"`

	// Azure defines azure specific settings
	Azure *AzurePlatformSpec `json:"azure,omitempty"`

	// PowerVS specifies configuration for clusters running on IBMCloud Power VS Service.
	// This field is immutable. Once set, It can't be changed.
	//
	// +optional
	// +immutable
	PowerVS *PowerVSPlatformSpec `json:"powervs,omitempty"`
}

// AWSPlatformSpec specifies configuration for clusters running on Amazon Web Services.
type AWSPlatformInfraSpec struct {
	// Region is the AWS region in which the cluster resides. This configures the
	// OCP control plane cloud integrations, and is used by NodePool to resolve
	// the correct boot AMI for a given release.
	// HostedCluster.spec.platform.aws.region
	//
	// +immutable
	Region string `json:"region"`

	// VPC is the VPC to use for control plane cloud resources.
	// HostedCluster.spec.platform.aws.cloudProviderConfig.vpc
	//
	// +optional
	// +immutable
	VPC string `json:"vpc,omitempty"`

	// RolesRef contains references to various AWS IAM roles required to enable
	// integrations such as OIDC.
	// HostedCluster.spec.platform.aws.rolesRef
	//
	// +optional
	// +immutable
	RolesRef *AWSRolesRef `json:"rolesRef,omitempty"`

	// SecurityGroups is an optional set of security groups to associate with node
	// instances. One of more of the security groups can be used with nodePool resources
	// NodePool.spec.platform.aws.securityGroups[]
	//
	// +optional
	SecurityGroups []AWSResourceReference `json:"securityGroups,omitempty"`

	// Zones are availability zones in an AWS region.
	// An AWS subnet is created in each zone. The info is then used to populate
	// HostedCluster.spec.platform.aws.cloudProviderConfig.zone
	// HostedCluster.spec.platform.aws.cloudProviderConfig.subnet
	// NodePool.spec.platform.aws.subnet.id
	//
	// +optional
	Zones []AWSZoneAndSubnet `json:"zones,omitempty"`

	// Not tracked from CLI
	// DNS support on VPC
	// DNS hostenames on VPC
	// DHCP options with vpc. dopt-......
	// Internet gateway
	// Attach internet gateway
	// security group
	// ingress rules
	// elastic IP for NAT gateway
	// NAT gateway
	// route table
	// route to NAT
	// subnet to route association
	// route to internet gateway
	// s3 VPC endpoint
}

type AWSZoneAndSubnet struct {
	// Subnet will be created if value is empty in the specified zone
	//
	// +optional
	Subnet *AWSResourceReference `json:"subnet,omitempty"`

	// Zone is the availability zone to be used, a subnet will be created if one is not provided.
	// The availability zones must be a memeber of the spec.platform.aws.region
	//
	Zone string `json:"zone"`
}

// HostedClusterInfrastructure defines the observed state of HostedClusterInfrastructure
type HostedClusterInfrastructureStatus struct {
	// Track the conditions for each step in the desired curation that is being
	// executed as a job
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	//Show which phase of curation is currently being processed
	// +kubebuilder:default=init
	Phase CurrentPhase `json:"phase,omitempty"`
}

// +genclient

// HostedClusterInfrastructure is the primary representation of a HyperShift cluster's infrastructure.
// It encapsulates resource that can be created orthogonal to the control plane. Creating a HostedClusterInfrastructure
// results in a set of provider resources that can be consumed by HostedCluster (hypershift-operator).
// This is not required for HostedCluster, but allows required infrastructure to be managed and staged,
// independent of HostClusters and NodePools
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=hostedclusterinfrastructure,shortName=hci;hcis,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".spec.platform.type",description="Infrastructure type"
// +kubebuilder:printcolumn:name="INFRA",type="string",JSONPath=".status.conditions[?(@.type==\"PlatformInfrastructureConfigured\")].reason",description="Reason"
// +kubebuilder:printcolumn:name="IAM",type="string",JSONPath=".status.conditions[?(@.type==\"PlatformIAMConfigured\")].reason",description="Reason"
// +kubebuilder:printcolumn:name="PROVIDER REF",type="string",JSONPath=".status.conditions[?(@.type==\"ProviderSecretConfigured\")].reason",description="Reason"
// +kubebuilder:printcolumn:name="PHASE",type="string",JSONPath=".status.phase",description="current phase"
type HostedClusterInfrastructure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired behavior of the HostedCluster.
	Spec HostedClusterInfrastructureSpec `json:"spec,omitempty"`

	// Status is the latest observed status of the HostedCluster.
	Status HostedClusterInfrastructureStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// HostedClusterList contains a list of HostedCluster
type HostedClusterInfrastructureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostedCluster `json:"items"`
}
