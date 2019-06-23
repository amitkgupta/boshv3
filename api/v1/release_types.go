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

	"github.com/amitkgupta/boshv3/remote-clients"
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

// ReleaseStatus defines the observed state of Release
type ReleaseStatus struct {
	Warning      string      `json:"warning"`
	OriginalSpec ReleaseSpec `json:"originalSpec"`
}

// +kubebuilder:object:root=true

// Release is the Schema for the releases API
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

func (r *Release) BeingDeleted() bool {
	return !r.ObjectMeta.DeletionTimestamp.IsZero()
}

var releaseFinalizer = strings.Join([]string{"release", finalizerBase}, ".")

func (r *Release) hasFinalizer() bool {
	return containsString(r.ObjectMeta.Finalizers, releaseFinalizer)
}

func (r *Release) EnsureFinalizer() bool {
	changed := !r.hasFinalizer()
	r.ObjectMeta.Finalizers = append(r.ObjectMeta.Finalizers, releaseFinalizer)
	return changed
}

func (r *Release) EnsureNoFinalizer() bool {
	changed := r.hasFinalizer()
	r.ObjectMeta.Finalizers = removeString(r.ObjectMeta.Finalizers, releaseFinalizer)
	return changed
}

func (r *Release) PrepareToSave() (needsStatusUpdate bool) {
	originalSpec := r.Status.OriginalSpec

	if originalSpec.ReleaseName == "" {
		r.Status.OriginalSpec = r.Spec
		needsStatusUpdate = true
	} else {
		mutated := r.Spec.ReleaseName != originalSpec.ReleaseName ||
			r.Spec.Version != originalSpec.Version ||
			r.Spec.URL != originalSpec.URL ||
			r.Spec.SHA1 != originalSpec.SHA1

		if mutated && r.Status.Warning == "" {
			r.Status.Warning = resourceMutationWarning
			needsStatusUpdate = true
		} else if !mutated && r.Status.Warning != "" {
			r.Status.Warning = ""
			needsStatusUpdate = true
		}
	}

	return
}

func (r *Release) CreateUnlessExists(bc remoteclients.BOSHClient) error {
	releaseSpec := r.Status.OriginalSpec

	if present, err := bc.HasRelease(
		releaseSpec.ReleaseName,
		releaseSpec.Version,
	); err != nil {
		return err
	} else if !present {
		return bc.UploadRelease(
			releaseSpec.URL,
			releaseSpec.SHA1,
		)
	}

	return nil
}

func (r *Release) DeleteIfExists(bc remoteclients.BOSHClient) error {
	releaseSpec := r.Status.OriginalSpec

	if present, err := bc.HasRelease(
		releaseSpec.ReleaseName,
		releaseSpec.Version,
	); err != nil {
		return err
	} else if present {
		return bc.DeleteRelease(
			releaseSpec.ReleaseName,
			releaseSpec.Version,
		)
	}

	return nil
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
