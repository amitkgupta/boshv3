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

// TeamSpec defines the desired state of Team
type TeamSpec struct {
	Director string `json:"director"`
}

// TeamStatus defines the observed state of Team
type TeamStatus struct {
	Warning          string `json:"warning"`
	OriginalDirector string `json:"original_director"`
	SecretNamespace  string `json:"secret_namespace"`
	Available        bool   `json:"available"`
}

// +kubebuilder:object:root=true

// Team is the Schema for the teams API
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

func (t *Team) BeingDeleted() bool {
	return !t.ObjectMeta.DeletionTimestamp.IsZero()
}

var teamFinalizer = strings.Join([]string{"team", finalizerBase}, ".")

func (t *Team) hasFinalizer() bool {
	return containsString(t.ObjectMeta.Finalizers, teamFinalizer)
}

func (t *Team) EnsureFinalizer() bool {
	changed := !t.hasFinalizer()
	t.ObjectMeta.Finalizers = append(t.ObjectMeta.Finalizers, teamFinalizer)
	return changed
}

func (t *Team) EnsureNoFinalizer() bool {
	changed := t.hasFinalizer()
	t.ObjectMeta.Finalizers = removeString(t.ObjectMeta.Finalizers, teamFinalizer)
	return changed
}

func (t *Team) PrepareToSave(secretNamespace string) (needsStatusUpdate bool) {
	originalDirector := t.Status.OriginalDirector

	if originalDirector == "" || t.Status.SecretNamespace == "" {
		t.Status.OriginalDirector = t.Spec.Director
		t.Status.SecretNamespace = secretNamespace
		needsStatusUpdate = true
	} else {
		if t.Spec.Director != originalDirector && t.Status.Warning == "" {
			t.Status.Warning = resourceMutationWarning
			needsStatusUpdate = true
		} else if t.Spec.Director == originalDirector && t.Status.Warning != "" {
			t.Status.Warning = ""
			needsStatusUpdate = true
		}
	}

	return
}

func (t *Team) ClientName() string {
	return standardName(t.ObjectMeta.Name, t.ObjectMeta.Namespace)
}

func (t *Team) SecretName() string {
	return standardName(t.ObjectMeta.Name, t.ObjectMeta.Namespace)
}

func (t *Team) SecretNamespace() string {
	return t.Status.SecretNamespace
}

func (t *Team) CreateUnlessExists(uc remoteclients.UAAClient, secretData string) error {
	if present, err := uc.HasClient(t.ClientName()); err != nil {
		return err
	} else if !present {
		if err := uc.CreateClient(
			t.ClientName(),
			secretData,
			[]string{"bosh.admin"}, // TODO: needed to delete releases, etc.
		); err != nil {
			return err
		}
	}

	t.Status.Available = true

	return nil
}

func (t *Team) DeleteIfExists(uc remoteclients.UAAClient) error {
	if present, err := uc.HasClient(t.ClientName()); err != nil {
		return err
	} else if present {
		return uc.DeleteClient(t.ClientName())
	}

	return nil
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}
