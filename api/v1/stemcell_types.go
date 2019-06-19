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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
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

func (s *Stemcell) BeingDeleted() bool {
	return !s.ObjectMeta.DeletionTimestamp.IsZero()
}

var stemcellFinalizer = strings.Join([]string{"stemcell", finalizerBase}, ".")

func (s *Stemcell) HasFinalizer() bool {
	return containsString(s.ObjectMeta.Finalizers, stemcellFinalizer)
}

func (s *Stemcell) EnsureFinalizer() bool {
	hadFinalizer := s.HasFinalizer()
	s.ObjectMeta.Finalizers = append(s.ObjectMeta.Finalizers, stemcellFinalizer)
	return !hadFinalizer
}

func (s *Stemcell) RemoveFinalizer() {
	s.ObjectMeta.Finalizers = removeString(s.ObjectMeta.Finalizers, stemcellFinalizer)
}

func (s *Stemcell) SaveOriginalSpec() (bool, bool) {
	originalSpec := s.Status.OriginalSpec

	saved := originalSpec.StemcellName == ""
	mutated := false

	if saved {
		s.Status.OriginalSpec = s.Spec
	} else {
		mutated = s.Spec.StemcellName != originalSpec.StemcellName ||
			s.Spec.Version != originalSpec.Version ||
			s.Spec.URL != originalSpec.URL ||
			s.Spec.SHA1 != originalSpec.SHA1
	}

	return saved, mutated
}

func (s *Stemcell) EnsureWarning() bool {
	changed := s.Status.Warning == ""
	s.Status.Warning = resourceMutationWarning
	return changed
}

func (s *Stemcell) EnsureNoWarning() bool {
	changed := s.Status.Warning != ""
	s.Status.Warning = ""
	return changed
}

func (s *Stemcell) EnsureAbsentFromDirector() bool {
	changed := s.Status.PresentOnDirector
	s.Status.PresentOnDirector = false
	return changed
}

func (s *Stemcell) EnsurePresentOnDirector() bool {
	changed := !s.Status.PresentOnDirector
	s.Status.PresentOnDirector = true
	return changed
}

func (s *Stemcell) PresentOnDirector(d boshdir.Director) (bool, error) {
	originalSpec := s.Status.OriginalSpec
	return d.HasStemcell(originalSpec.StemcellName, originalSpec.Version)
}

func (s *Stemcell) UploadToDirector(d boshdir.Director) error {
	originalSpec := s.Status.OriginalSpec
	return d.UploadStemcellURL(originalSpec.URL, originalSpec.SHA1, false)
}

func (s *Stemcell) DeleteFromDirector(d boshdir.Director) error {
	if present, err := s.PresentOnDirector(d); err != nil {
		return err
	} else if !present {
		return nil
	}

	originalSpec := s.Status.OriginalSpec
	if stemcell, err := d.FindStemcell(boshdir.NewStemcellSlug(originalSpec.StemcellName, originalSpec.Version)); err != nil {
		return err
	} else {
		return stemcell.Delete(false)
	}
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
