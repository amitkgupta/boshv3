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

func (r *Release) BeingDeleted() bool {
	return !r.ObjectMeta.DeletionTimestamp.IsZero()
}

var releaseFinalizer = strings.Join([]string{"release", finalizerBase}, ".")

func (r *Release) HasFinalizer() bool {
	return containsString(r.ObjectMeta.Finalizers, releaseFinalizer)
}

func (r *Release) EnsureFinalizer() bool {
	hadFinalizer := r.HasFinalizer()
	r.ObjectMeta.Finalizers = append(r.ObjectMeta.Finalizers, releaseFinalizer)
	return !hadFinalizer
}

func (r *Release) RemoveFinalizer() {
	r.ObjectMeta.Finalizers = removeString(r.ObjectMeta.Finalizers, releaseFinalizer)
}

func (r *Release) SaveOriginalSpec() (bool, bool) {
	originalSpec := r.Status.OriginalSpec

	saved := originalSpec.ReleaseName == ""
	mutated := false

	if saved {
		r.Status.OriginalSpec = r.Spec
	} else {
		mutated = r.Spec.ReleaseName != originalSpec.ReleaseName ||
			r.Spec.Version != originalSpec.Version ||
			r.Spec.URL != originalSpec.URL ||
			r.Spec.SHA1 != originalSpec.SHA1
	}

	return saved, mutated
}

func (r *Release) EnsureWarning() bool {
	changed := r.Status.Warning == ""
	r.Status.Warning = resourceMutationWarning
	return changed
}

func (r *Release) EnsureNoWarning() bool {
	changed := r.Status.Warning != ""
	r.Status.Warning = ""
	return changed
}

func (r *Release) EnsureAbsentFromDirector() bool {
	changed := r.Status.PresentOnDirector
	r.Status.PresentOnDirector = false
	return changed
}

func (r *Release) EnsurePresentOnDirector() bool {
	changed := !r.Status.PresentOnDirector
	r.Status.PresentOnDirector = true
	return changed
}

func (r *Release) PresentOnDirector(d boshdir.Director) (bool, error) {
	originalSpec := r.Status.OriginalSpec
	return d.HasRelease(originalSpec.ReleaseName, originalSpec.Version, boshdir.OSVersionSlug{})
}

func (r *Release) UploadToDirector(d boshdir.Director) error {
	originalSpec := r.Status.OriginalSpec
	return d.UploadReleaseURL(originalSpec.URL, originalSpec.SHA1, false, false)
}

func (r *Release) DeleteFromDirector(d boshdir.Director) error {
	if present, err := r.PresentOnDirector(d); err != nil {
		return err
	} else if !present {
		return nil
	}

	originalSpec := r.Status.OriginalSpec
	if release, err := d.FindRelease(boshdir.NewReleaseSlug(originalSpec.ReleaseName, originalSpec.Version)); err != nil {
		return err
	} else {
		return release.Delete(false)
	}
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
