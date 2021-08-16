/*
Copyright 2021.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//DatabaseSize int32 `json:"databaseSize"`

// VisitorsAppSpec defines the desired state of VisitorsApp
//+k8s:openapi-gen=true
type VisitorsAppSpec struct {
	//+kubebuilder:validation:Minimum=1
	BackendSize int32 `json:"backendSize"`

	//+kubebuilder:validation:Minimum=30000
	//+kubebuilder:validation:Maximum=32767
	BackendServiceNodePort int32 `json:"backendServiceNodePort"`

	FrontendTitle string `json:"frontendTitle"`

	//+kubebuilder:validation:Minimum=1
	FrontendSize int32 `json:"frontendSize"`

	//+kubebuilder:validation:Minimum=30000
	//+kubebuilder:validation:Maximum=32767
	FrontendServiceNodePort int32 `json:"frontendServiceNodePort"`
}

// VisitorsAppStatus defines the observed state of VisitorsApp
//+k8s:openapi-gen=true
type VisitorsAppStatus struct {
	BackendImage  string `json:"backendImage,omitempty"`
	FrontendImage string `json:"frontendImage,omitempty"`
}

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VisitorsApp is the Schema for the visitorsapps API
//+k8s:openapi-gen=true
//+kubebuilder:subresource:status
type VisitorsApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VisitorsAppSpec   `json:"spec,omitempty"`
	Status VisitorsAppStatus `json:"status,omitempty"`
}

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true

// VisitorsAppList contains a list of VisitorsApp
type VisitorsAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VisitorsApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VisitorsApp{}, &VisitorsAppList{})
}
