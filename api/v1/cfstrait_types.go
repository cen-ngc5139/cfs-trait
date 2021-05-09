/*


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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CfsTraitSpec defines the desired state of CfsTrait
type CfsTraitSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	AppName   string   `json:"appName"`
	Force     bool     `json:"force,omitempty"`
	LabelKey  string   `json:"labelKey"`
	IsAllPods bool     `json:"isAllPods,omitempty"`
	Pods      []string `json:"pods,omitempty"`
	Namespace string   `json:"namespace"`
	Period    int32    `json:"period"`
	Quota     int32    `json:"quota"`
}

// CfsTraitStatus defines the observed state of CfsTrait
type CfsTraitStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Nodes int64 `json:"nodes,omitempty"`

	UpdatedNodes int64 `json:"updatedNodes,omitempty"`

	Conditions []NodeCondition `json:"conditions,omitempty"`

	// The generation observed by the appConfig controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LatestRevision of component
	// +optional
	LatestRevision *Revision `json:"latestRevision,omitempty"`
}

// NodeCondition DeploymentCondition describes the state of a deployment at a certain point.
type NodeCondition struct {
	NodeName string                 `json:"nodeName"`
	Status   corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" `
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// Revision has name and revision number
type Revision struct {
	Name     string `json:"name"`
	Revision int64  `json:"revision"`
}

// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ct
// +kubebuilder:categories=cfstraits

// CfsTrait is the Schema for the cfstraits API
type CfsTrait struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CfsTraitSpec   `json:"spec,omitempty"`
	Status CfsTraitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CfsTraitList contains a list of CfsTrait
type CfsTraitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CfsTrait `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CfsTrait{}, &CfsTraitList{})
}
