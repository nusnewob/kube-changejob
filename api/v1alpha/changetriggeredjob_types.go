/*
Copyright 2025 Bowen Sun.

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

package v1alpha

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ChangeTriggeredJobSpec defines the desired state of ChangeTriggeredJob
type ChangeTriggeredJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// jobTemplate defines the job that will be created when executing a Job.
	// +required
	JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate"`

	// list of resources to watch
	// +required
	Resources []ResourceReference `json:"resources"`

	// Trigger condition, job triggers when All or Any watched resource changes
	// +optional
	// +default:value="Any"
	Condition TriggerCondition `json:"condition"`

	// Optional: cooldown period between triggers
	// +optional
	// +default:value="60s"
	Cooldown metav1.Duration `json:"cooldown,omitempty"`
}

// Watched Resource object
type ResourceReference struct {
	// API group of the resource, e.g., apps/v1, example.io/v1beta
	// +required
	APIVersion string `json:"apiVersion"`

	// Kind of the Kubernetes resource, e.g., ConfigMap, Secret
	// +required
	Kind string `json:"kind"`

	// Name of the resource
	// +required
	Name string `json:"name"`

	// Namespace of the resource (optional for cluster-scoped resources)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Optional: JSON Path of fields to watch within the resource
	// +optional
	// +kubebuilder:default={"*"}
	Fields []string `json:"fields,omitempty"`
}

// Define trigger conditions
// +kubebuilder:validation:Enum:=All;Any
type TriggerCondition string

const (
	TriggerConditionAll TriggerCondition = "All"
	TriggerConditionAny TriggerCondition = "Any"
)

// ChangeTriggeredJobStatus defines the observed state of ChangeTriggeredJob.
type ChangeTriggeredJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the ChangeTriggeredJob resource.
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
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Last change hash
	// +optional
	ResourceHashes []ResourceReferenceStatus `json:"resourceHashes,omitempty"`

	// Last Job triggered time
	// +optional
	LastTriggeredTime *metav1.Time `json:"lastTriggeredTime,omitempty"`

	// Last Job name
	// +optional
	LastJobName string `json:"lastJobName,omitempty"`

	// Last Job status
	// +optional
	LastJobStatus JobState `json:"lastJobStatus,omitempty"`
}

// Watched ResouceHash object
type ResourceReferenceStatus struct {
	// API group of the resource, e.g., apps/v1, example.io/v1beta
	// +optional
	APIVersion string `json:"apiVersion"`

	// Kind of the Kubernetes resource, e.g., ConfigMap, Secret
	// +optional
	Kind string `json:"kind"`

	// Name of the resource
	// +optional
	Name string `json:"name"`

	// Namespace of the resource (optional for cluster-scoped resources)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Optional: fields to watch within the resource
	// +optional
	Fields []ResourceFieldHash `json:"fields,omitempty"`
}

type ResourceFieldHash struct {
	Field    string `json:"field"`
	LastHash string `json:"hash"`
}

// Define last job state
// +kubebuilder:validation:Enum:=Active;Succeeded;Failed
type JobState string

const (
	JobStateActive    JobState = "Active"
	JobStateSucceeded JobState = "Succeeded"
	JobStateFailed    JobState = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ChangeTriggeredJob is the Schema for the changetriggeredjobs API
type ChangeTriggeredJob struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of ChangeTriggeredJob
	// +required
	Spec ChangeTriggeredJobSpec `json:"spec"`

	// status defines the observed state of ChangeTriggeredJob
	// +optional
	Status ChangeTriggeredJobStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ChangeTriggeredJobList contains a list of ChangeTriggeredJob
type ChangeTriggeredJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ChangeTriggeredJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChangeTriggeredJob{}, &ChangeTriggeredJobList{})
}
