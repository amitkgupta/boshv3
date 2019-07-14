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

// BaseImageSpec defines the desired state of BaseImage
// +kubebuilder:subresource:status
type BaseImageSpec struct {
	BaseImageName string `json:"baseImageName"`
	Version       string `json:"version"`
	URL           string `json:"url"`
	SHA1          string `json:"sha1"`
}

// BaseImageStatus defines the observed state of BaseImage
type BaseImageStatus struct {
	Warning      string        `json:"warning"`
	OriginalSpec BaseImageSpec `json:"originalSpec"`
	Available    bool          `json:"available"`
}

// +kubebuilder:object:root=true

// BaseImage is the Schema for the baseImages API
type BaseImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BaseImageSpec   `json:"spec,omitempty"`
	Status BaseImageStatus `json:"status,omitempty"`
}

func (s BaseImage) BeingDeleted() bool {
	return !s.GetDeletionTimestamp().IsZero()
}

var baseImageFinalizer = strings.Join([]string{"base-image", finalizerBase}, ".")

func (s BaseImage) hasFinalizer() bool {
	return containsString(s.GetFinalizers(), baseImageFinalizer)
}

func (s *BaseImage) EnsureFinalizer() bool {
	changed := !s.hasFinalizer()
	s.SetFinalizers(append(s.GetFinalizers(), baseImageFinalizer))
	return changed
}

func (s *BaseImage) EnsureNoFinalizer() bool {
	changed := s.hasFinalizer()
	s.SetFinalizers(removeString(s.GetFinalizers(), baseImageFinalizer))
	return changed
}

func (s *BaseImage) PrepareToSave() (needsStatusUpdate bool) {
	originalSpec := s.Status.OriginalSpec

	if originalSpec.BaseImageName == "" {
		s.Status.OriginalSpec = s.Spec
		needsStatusUpdate = true
	} else {
		mutated := s.Spec.BaseImageName != originalSpec.BaseImageName ||
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

func (s *BaseImage) CreateUnlessExists(
	bc remoteclients.BOSHClient,
	_ context.Context,
	_ client.Client,
) error {
	baseImageSpec := s.Status.OriginalSpec

	if present, err := bc.HasBaseImage(
		baseImageSpec.BaseImageName,
		baseImageSpec.Version,
	); err != nil {
		return err
	} else if !present {
		if err := bc.UploadBaseImage(
			baseImageSpec.URL,
			baseImageSpec.SHA1,
		); err != nil {
			return err
		}
	}

	s.Status.Available = true

	return nil
}

func (s BaseImage) DeleteIfExists(bc remoteclients.BOSHClient) error {
	baseImageSpec := s.Status.OriginalSpec

	if present, err := bc.HasBaseImage(
		baseImageSpec.BaseImageName,
		baseImageSpec.Version,
	); err != nil {
		return err
	} else if present {
		return bc.DeleteBaseImage(
			baseImageSpec.BaseImageName,
			baseImageSpec.Version,
		)
	}

	return nil
}

// +kubebuilder:object:root=true

// BaseImageList contains a list of BaseImage
type BaseImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BaseImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BaseImage{}, &BaseImageList{})
}
