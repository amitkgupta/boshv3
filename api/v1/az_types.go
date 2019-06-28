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
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/amitkgupta/boshv3/remote-clients"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AZSpec defines the desired state of AZ
type AZSpec struct {
	CloudProperties *runtime.RawExtension `json:"cloud_properties"`
}

// AZStatus defines the observed state of AZ
type AZStatus struct {
	Warning                 string                `json:"warning"`
	OriginalCloudProperties *runtime.RawExtension `json:"cloud_properties"`
	Available               bool                  `json:"available"`
}

// +kubebuilder:object:root=true

// AZ is the Schema for the azs API
type AZ struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AZSpec   `json:"spec,omitempty"`
	Status AZStatus `json:"status,omitempty"`
}

func (a *AZ) BeingDeleted() bool {
	return !a.ObjectMeta.DeletionTimestamp.IsZero()
}

var azFinalizer = strings.Join([]string{"az", finalizerBase}, ".")

func (a *AZ) hasFinalizer() bool {
	return containsString(a.ObjectMeta.Finalizers, azFinalizer)
}

func (a *AZ) EnsureFinalizer() bool {
	changed := !a.hasFinalizer()
	a.ObjectMeta.Finalizers = append(a.ObjectMeta.Finalizers, azFinalizer)
	return changed
}

func (a *AZ) EnsureNoFinalizer() bool {
	changed := a.hasFinalizer()
	a.ObjectMeta.Finalizers = removeString(a.ObjectMeta.Finalizers, azFinalizer)
	return changed
}

func (a *AZ) PrepareToSave() (needsStatusUpdate bool) {
	originalCloudProperties := a.Status.OriginalCloudProperties

	if originalCloudProperties == nil {
		a.Status.OriginalCloudProperties = a.Spec.CloudProperties
		needsStatusUpdate = true
	} else {
		mutated := a.Spec.CloudProperties.String() != originalCloudProperties.String()

		if mutated && a.Status.Warning == "" {
			a.Status.Warning = resourceMutationWarning
			needsStatusUpdate = true
		} else if !mutated && a.Status.Warning != "" {
			a.Status.Warning = ""
			needsStatusUpdate = true
		}
	}

	return
}

func (a *AZ) InternalName() string {
	return strings.Join([]string{
		"az",
		a.ObjectMeta.Namespace,
		a.ObjectMeta.Name,
	}, "-")
}

func (a *AZ) CreateUnlessExists(bc remoteclients.BOSHClient) error {
	if err := bc.CreateAZ(
		a.InternalName(),
		a.Status.OriginalCloudProperties,
	); err != nil {
		return err
	}

	a.Status.Available = true

	return nil
}

func (a *AZ) DeleteIfExists(bc remoteclients.BOSHClient) error {
	return bc.DeleteAZ(a.InternalName())
}

// +kubebuilder:object:root=true

// AZList contains a list of AZ
type AZList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AZ `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AZ{}, &AZList{})
}
