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

// StemcellSpec defines the desired state of Stemcell
// +kubebuilder:subresource:status
type StemcellSpec struct {
	StemcellName string `json:"stemcellName"`
	Version      string `json:"version"`
	URL          string `json:"url"`
	SHA1         string `json:"sha1"`
}

func (ss StemcellSpec) Empty() bool {
	return ss.StemcellName == "" &&
		ss.Version == "" &&
		ss.URL == "" &&
		ss.SHA1 == ""
}

func (ss1 StemcellSpec) Matches(ss2 StemcellSpec) bool {
	return ss1.StemcellName == ss2.StemcellName &&
		ss1.Version == ss2.Version &&
		ss1.URL == ss2.URL &&
		ss1.SHA1 == ss2.SHA1
}

// StemcellStatus defines the observed state of Stemcell
type StemcellStatus struct {
	Warning           string       `json:"warning"`
	OriginalSpec      StemcellSpec `json:"originalSpec"`
	PresentOnDirector bool         `json:"presentOnDirector"`
}

// +kubebuilder:object:root=true

// Stemcell is the Schema for the stemcells API
type Stemcell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StemcellSpec   `json:"spec,omitempty"`
	Status StemcellStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StemcellList contains a list of Stemcell
type StemcellList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stemcell `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stemcell{}, &StemcellList{})
}
