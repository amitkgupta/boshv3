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
	"encoding/json"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/amitkgupta/boshv3/remote-clients"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VMExtensionSpec defines the desired state of VMExtension
// +kubebuilder:subresource:status
type VMExtensionSpec struct {
	CloudProperties json.RawMessage `json:"cloud_properties"`
}

// VMExtensionStatus defines the observed state of VMExtension
type VMExtensionStatus struct {
	Warning                 string          `json:"warning"`
	OriginalCloudProperties json.RawMessage `json:"cloud_properties"`
	Available               bool            `json:"available"`
}

// +kubebuilder:object:root=true

// VMExtension is the Schema for the vmextensions API
type VMExtension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMExtensionSpec   `json:"spec,omitempty"`
	Status VMExtensionStatus `json:"status,omitempty"`
}

func (v *VMExtension) BeingDeleted() bool {
	return !v.ObjectMeta.DeletionTimestamp.IsZero()
}

var vmExtensionFinalizer = strings.Join([]string{"vm-extension", finalizerBase}, ".")

func (v *VMExtension) hasFinalizer() bool {
	return containsString(v.ObjectMeta.Finalizers, vmExtensionFinalizer)
}

func (v *VMExtension) EnsureFinalizer() bool {
	changed := !v.hasFinalizer()
	v.ObjectMeta.Finalizers = append(v.ObjectMeta.Finalizers, vmExtensionFinalizer)
	return changed
}

func (v *VMExtension) EnsureNoFinalizer() bool {
	changed := v.hasFinalizer()
	v.ObjectMeta.Finalizers = removeString(v.ObjectMeta.Finalizers, vmExtensionFinalizer)
	return changed
}

func (v *VMExtension) PrepareToSave() (needsStatusUpdate bool) {
	originalCloudProperties := v.Status.OriginalCloudProperties

	if originalCloudProperties == nil {
		v.Status.OriginalCloudProperties = v.Spec.CloudProperties
		needsStatusUpdate = true
	} else {
		mutated := string(v.Spec.CloudProperties) != string(originalCloudProperties)

		if mutated && v.Status.Warning == "" {
			v.Status.Warning = resourceMutationWarning
			needsStatusUpdate = true
		} else if !mutated && v.Status.Warning != "" {
			v.Status.Warning = ""
			needsStatusUpdate = true
		}
	}

	return
}

func (v *VMExtension) InternalName() string {
	return strings.Join([]string{
		"vmextension",
		v.ObjectMeta.Namespace,
		v.ObjectMeta.Name,
	}, "-")
}

func (v *VMExtension) CreateUnlessExists(bc remoteclients.BOSHClient) error {
	if present, err := bc.HasVMExtension(v.InternalName()); err != nil {
		return err
	} else if !present {
		if err := bc.CreateVMExtension(
			v.InternalName(),
			v.Status.OriginalCloudProperties,
		); err != nil {
			return err
		}
	}

	v.Status.Available = true

	return nil
}

func (v *VMExtension) DeleteIfExists(bc remoteclients.BOSHClient) error {
	if present, err := bc.HasVMExtension(v.InternalName()); err != nil {
		return err
	} else if present {
		return bc.DeleteVMExtension(v.InternalName())
	}

	return nil
}

// +kubebuilder:object:root=true

// VMExtensionList contains a list of VMExtension
type VMExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMExtension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VMExtension{}, &VMExtensionList{})
}
