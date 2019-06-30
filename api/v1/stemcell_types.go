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
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/amitkgupta/boshv3/remote-clients"
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
	Warning      string       `json:"warning"`
	OriginalSpec StemcellSpec `json:"originalSpec"`
	Available    bool         `json:"available"`
}

// +kubebuilder:object:root=true

// Stemcell is the Schema for the stemcells API
type Stemcell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StemcellSpec   `json:"spec,omitempty"`
	Status StemcellStatus `json:"status,omitempty"`
}

func (s Stemcell) BeingDeleted() bool {
	return !s.ObjectMeta.DeletionTimestamp.IsZero()
}

var stemcellFinalizer = strings.Join([]string{"stemcell", finalizerBase}, ".")

func (s Stemcell) hasFinalizer() bool {
	return containsString(s.ObjectMeta.Finalizers, stemcellFinalizer)
}

func (s *Stemcell) EnsureFinalizer() bool {
	changed := !s.hasFinalizer()
	s.ObjectMeta.Finalizers = append(s.ObjectMeta.Finalizers, stemcellFinalizer)
	return changed
}

func (s *Stemcell) EnsureNoFinalizer() bool {
	changed := s.hasFinalizer()
	s.ObjectMeta.Finalizers = removeString(s.ObjectMeta.Finalizers, stemcellFinalizer)
	return changed
}

func (s *Stemcell) PrepareToSave() (needsStatusUpdate bool) {
	originalSpec := s.Status.OriginalSpec

	if originalSpec.StemcellName == "" {
		s.Status.OriginalSpec = s.Spec
		needsStatusUpdate = true
	} else {
		mutated := s.Spec.StemcellName != originalSpec.StemcellName ||
			s.Spec.Version != originalSpec.Version ||
			s.Spec.URL != originalSpec.URL ||
			s.Spec.SHA1 != originalSpec.SHA1

		if mutated && s.Status.Warning == "" {
			s.Status.Warning = resourceMutationWarning
			needsStatusUpdate = true
		} else if !mutated && s.Status.Warning != "" {
			s.Status.Warning = ""
			needsStatusUpdate = true
		}
	}

	return
}

func (s *Stemcell) CreateUnlessExists(
	bc remoteclients.BOSHClient,
	_ context.Context,
	_ client.Client,
) error {
	stemcellSpec := s.Status.OriginalSpec

	if present, err := bc.HasStemcell(
		stemcellSpec.StemcellName,
		stemcellSpec.Version,
	); err != nil {
		return err
	} else if !present {
		if err := bc.UploadStemcell(
			stemcellSpec.URL,
			stemcellSpec.SHA1,
		); err != nil {
			return err
		}
	}

	s.Status.Available = true

	return nil
}

func (s Stemcell) DeleteIfExists(bc remoteclients.BOSHClient) error {
	stemcellSpec := s.Status.OriginalSpec

	if present, err := bc.HasStemcell(
		stemcellSpec.StemcellName,
		stemcellSpec.Version,
	); err != nil {
		return err
	} else if present {
		return bc.DeleteStemcell(
			stemcellSpec.StemcellName,
			stemcellSpec.Version,
		)
	}

	return nil
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
