/*
Copyright 2019 Amit Kumar Gupta.

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

// ReleaseSpec defines the desired state of Release
// +kubebuilder:subresource:status
type ReleaseSpec struct {
	ReleaseName string `json:"releaseName"`
	Version     string `json:"version"`
	URL         string `json:"url"`
	SHA1        string `json:"sha1"`
}

func (rs ReleaseSpec) Empty() bool {
	return rs.ReleaseName == "" &&
		rs.Version == "" &&
		rs.URL == "" &&
		rs.SHA1 == ""
}

func (rs1 ReleaseSpec) Matches(rs2 ReleaseSpec) bool {
	return rs1.ReleaseName == rs2.ReleaseName &&
		rs1.Version == rs2.Version &&
		rs1.URL == rs2.URL &&
		rs1.SHA1 == rs2.SHA1
}

// ReleaseStatus defines the observed state of Release
type ReleaseStatus struct {
	Warning           string      `json:"warning"`
	OriginalSpec      ReleaseSpec `json:"originalSpec"`
	PresentOnDirector bool        `json:"presentOnDirector"`
}

// +kubebuilder:object:root=true

// Release is the Schema for the releases API
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
