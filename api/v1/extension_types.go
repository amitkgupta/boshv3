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

// ExtensionSpec defines the desired state of Extension
// +kubebuilder:subresource:status
type ExtensionSpec struct {
	CloudProperties *runtime.RawExtension `json:"cloud_properties"`
}

// ExtensionStatus defines the observed state of Extension
type ExtensionStatus struct {
	Warning                 string                `json:"warning"`
	OriginalCloudProperties *runtime.RawExtension `json:"cloud_properties"`
	Available               bool                  `json:"available"`
}

// +kubebuilder:object:root=true

// Extension is the Schema for the extensions API
type Extension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExtensionSpec   `json:"spec,omitempty"`
	Status ExtensionStatus `json:"status,omitempty"`
}

func (e Extension) BeingDeleted() bool {
	return !e.GetDeletionTimestamp().IsZero()
}

var extensionFinalizer = strings.Join([]string{"extension", finalizerBase}, ".")

func (e Extension) hasFinalizer() bool {
	return containsString(e.GetFinalizers(), extensionFinalizer)
}

func (e *Extension) EnsureFinalizer() bool {
	changed := !e.hasFinalizer()
	e.SetFinalizers(append(e.GetFinalizers(), extensionFinalizer))
	return changed
}

func (e *Extension) EnsureNoFinalizer() bool {
	changed := e.hasFinalizer()
	e.SetFinalizers(removeString(e.GetFinalizers(), extensionFinalizer))
	return changed
}

func (e *Extension) PrepareToSave() (needsStatusUpdate bool) {
	originalCloudProperties := e.Status.OriginalCloudProperties

	if originalCloudProperties == nil {
		e.Status.OriginalCloudProperties = e.Spec.CloudProperties
		needsStatusUpdate = true
	} else {
		mutated := e.Spec.CloudProperties.String() != originalCloudProperties.String()

		if mutated && e.Status.Warning == "" {
			e.Status.Warning = resourceMutationWarning
			needsStatusUpdate = true
		} else if !mutated && e.Status.Warning != "" {
			e.Status.Warning = ""
			needsStatusUpdate = true
		}
	}

	return
}

func (e Extension) InternalName() string {
	return strings.Join([]string{
		"extension",
		e.GetNamespace(),
		e.GetName(),
	}, "-")
}

func (e *Extension) CreateUnlessExists(
	bc remoteclients.BOSHClient,
	_ context.Context,
	_ client.Client,
) error {
	if err := bc.CreateVMExtension(
		e.InternalName(),
		remoteclients.VMExtension{
			Name:            e.InternalName(),
			CloudProperties: e.Status.OriginalCloudProperties,
		},
	); err != nil {
		return err
	}

	e.Status.Available = true

	return nil
}

func (e Extension) DeleteIfExists(bc remoteclients.BOSHClient) error {
	return bc.DeleteVMExtension(e.InternalName())
}

// +kubebuilder:object:root=true

// ExtensionList contains a list of Extension
type ExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Extension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Extension{}, &ExtensionList{})
}
