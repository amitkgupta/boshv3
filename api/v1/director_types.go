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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DirectorSpec defines the desired state of Director
type DirectorSpec struct {
	URL             string `json:"url"`
	CACert          string `json:"ca_cert"`
	UAAURL          string `json:"uaa_url"`
	UAACACert       string `json:"uaa_ca_cert"`
	UAAClient       string `json:"uaa_client"`
	UAAClientSecret string `json:"uaa_client_secret"`
}

// DirectorStatus defines the observed state of Director
type DirectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// Director is the Schema for the directors API
type Director struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DirectorSpec   `json:"spec,omitempty"`
	Status DirectorStatus `json:"status,omitempty"`
}

func (d Director) Team() Team {
	return Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.teamName(),
			Namespace: d.GetNamespace(),
		},
		Spec: TeamSpec{
			Director: d.GetName(),
		},
	}
}

func (d Director) teamName() string {
	return strings.Join(
		[]string{
			"director",
			d.GetName(),
			"team",
		},
		"-",
	)
}

func (d Director) BeingDeleted() bool {
	return !d.GetDeletionTimestamp().IsZero()
}

var directorFinalizer = strings.Join([]string{"director", finalizerBase}, ".")

func (d Director) hasFinalizer() bool {
	return containsString(d.GetFinalizers(), directorFinalizer)
}

func (d *Director) EnsureFinalizer() bool {
	changed := !d.hasFinalizer()
	d.SetFinalizers(append(d.GetFinalizers(), directorFinalizer))
	return changed
}

func (d *Director) EnsureNoFinalizer() bool {
	changed := d.hasFinalizer()
	d.SetFinalizers(removeString(d.GetFinalizers(), directorFinalizer))
	return changed
}

// +kubebuilder:object:root=true

// DirectorList contains a list of Director
type DirectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Director `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Director{}, &DirectorList{})
}
