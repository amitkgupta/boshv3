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
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/amitkgupta/boshv3/remote-clients"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AZSpec defines the desired state of AZ
type AZSpec struct {
	CloudProperties *runtime.RawExtension `json:"cloud_properties,omitempty"`
}

// AZStatus defines the observed state of AZ
type AZStatus struct {
	ImmutableFieldsFrozen   bool                  `json:"immutable_fields_frozen"`
	Warning                 string                `json:"warning"`
	OriginalCloudProperties *runtime.RawExtension `json:"cloud_properties,omitempty"`
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

func (a AZ) BeingDeleted() bool {
	return !a.GetDeletionTimestamp().IsZero()
}

var azFinalizer = strings.Join([]string{"az", finalizerBase}, ".")

func (a AZ) hasFinalizer() bool {
	return containsString(a.GetFinalizers(), azFinalizer)
}

func (a *AZ) EnsureFinalizer() bool {
	changed := !a.hasFinalizer()
	a.SetFinalizers(append(a.GetFinalizers(), azFinalizer))
	return changed
}

func (a *AZ) EnsureNoFinalizer() bool {
	changed := a.hasFinalizer()
	a.SetFinalizers(removeString(a.GetFinalizers(), azFinalizer))
	return changed
}

func (a *AZ) PrepareToSave() bool {
	if !a.Status.ImmutableFieldsFrozen {
		a.Status.ImmutableFieldsFrozen = true
		a.Status.OriginalCloudProperties = a.Spec.CloudProperties
		return true
	}

	mutated := a.Spec.CloudProperties.String() != a.Status.OriginalCloudProperties.String()

	if mutated && a.Status.Warning == "" {
		a.Status.Warning = resourceMutationWarning
		return true
	} else if !mutated && a.Status.Warning != "" {
		a.Status.Warning = ""
		return true
	}

	return false
}

func (a AZ) InternalName() string {
	return strings.Join([]string{
		"az",
		a.GetNamespace(),
		a.GetName(),
	}, "-")
}

func (a *AZ) CreateUnlessExists(
	bc remoteclients.BOSHClient,
	_ context.Context,
	_ client.Client,
) error {
	if err := bc.CreateAZ(
		a.InternalName(),
		remoteclients.AZ{
			Name:            a.InternalName(),
			CloudProperties: a.Status.OriginalCloudProperties,
		},
	); err != nil {
		return err
	}

	a.Status.Available = true

	return nil
}

func (a AZ) DeleteIfExists(bc remoteclients.BOSHClient) error {
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
