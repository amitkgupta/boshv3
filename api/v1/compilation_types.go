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

// CompilationSpec defines the desired state of Compilation
type CompilationSpec struct {
	Replicas              int                   `json:"replicas"`
	AZCloudProperties     *runtime.RawExtension `json:"az_cloud_properties,omitempty"`
	CPU                   int                   `json:"cpu"`
	RAM                   int                   `json:"ram"`
	EphemeralDiskSize     int                   `json:"ephemeral_disk_size"`
	CloudProperties       *runtime.RawExtension `json:"cloud_properties,omitempty"`
	NetworkType           string                `json:"network_type"`
	SubnetRange           string                `json:"subnet_range"`
	SubnetGateway         string                `json:"subnet_gateway"`
	SubnetDNS             []string              `json:"subnet_dns"`
	SubnetReserved        []string              `json:"subnet_reserved,omitempty"`
	SubnetCloudProperties *runtime.RawExtension `json:"subnet_cloud_properties,omitempty"`
	Director              string                `json:"director"`
}

// CompilationStatus defines the observed state of Compilation
type CompilationStatus struct {
	Warning          string `json:"warning"`
	OriginalDirector string `json:"original_director"`
	Available        bool   `json:"available"`
}

// +kubebuilder:object:root=true

// Compilation is the Schema for the compilations API
type Compilation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CompilationSpec   `json:"spec,omitempty"`
	Status CompilationStatus `json:"status,omitempty"`
}

func (c Compilation) BeingDeleted() bool {
	return !c.GetDeletionTimestamp().IsZero()
}

var compilationFinalizer = strings.Join([]string{"compilation", finalizerBase}, ".")

func (c Compilation) hasFinalizer() bool {
	return containsString(c.GetFinalizers(), compilationFinalizer)
}

func (c *Compilation) EnsureFinalizer() bool {
	changed := !c.hasFinalizer()
	c.SetFinalizers(append(c.GetFinalizers(), compilationFinalizer))
	return changed
}

func (c *Compilation) EnsureNoFinalizer() bool {
	changed := c.hasFinalizer()
	c.SetFinalizers(removeString(c.GetFinalizers(), compilationFinalizer))
	return changed
}

func (c *Compilation) PrepareToSave() (needsStatusUpdate bool) {
	originalDirector := c.Status.OriginalDirector

	if originalDirector == "" {
		c.Status.OriginalDirector = c.Spec.Director
		needsStatusUpdate = true
	} else {
		mutated := originalDirector != c.Spec.Director

		if mutated && c.Status.Warning == "" {
			c.Status.Warning = resourceMutationWarning
			needsStatusUpdate = true
		} else if !mutated && c.Status.Warning != "" {
			c.Status.Warning = ""
			needsStatusUpdate = true
		}
	}

	return
}

func (c Compilation) InternalName() string {
	return strings.Join([]string{
		"compilation",
		c.GetNamespace(),
		c.GetName(),
	}, "-")
}

func (c *Compilation) CreateUnlessExists(
	bc remoteclients.BOSHClient,
	_ context.Context,
	_ client.Client,
) error {
	if err := bc.CreateCompilation(
		c.InternalName(),
		c.boshClientNetwork(),
		c.boshClientAZ(),
		c.boshClientCompilation(),
	); err != nil {
		return err
	}

	c.Status.Available = true

	return nil
}

func (c Compilation) boshClientAZ() remoteclients.AZ {
	return remoteclients.AZ{
		Name:            c.InternalName(),
		CloudProperties: c.Spec.AZCloudProperties,
	}
}

func (c Compilation) boshClientNetwork() remoteclients.Network {
	return remoteclients.Network{
		Name: c.InternalName(),
		Type: c.Spec.NetworkType,
		Subnets: []remoteclients.Subnet{remoteclients.Subnet{
			Range:           c.Spec.SubnetRange,
			Gateway:         c.Spec.SubnetGateway,
			DNS:             c.Spec.SubnetDNS,
			Reserved:        c.Spec.SubnetReserved,
			AZs:             []string{c.InternalName()},
			CloudProperties: c.Spec.SubnetCloudProperties,
		}},
	}
}

func (c Compilation) boshClientCompilation() remoteclients.Compilation {
	return remoteclients.Compilation{
		Workers:       c.Spec.Replicas,
		AZ:            c.InternalName(),
		OrphanWorkers: true,
		VMResources: remoteclients.VMResources{
			CPU:               c.Spec.CPU,
			RAM:               c.Spec.RAM,
			EphemeralDiskSize: c.Spec.EphemeralDiskSize,
		},
		CloudProperties:     c.Spec.CloudProperties,
		Network:             c.InternalName(),
		ReuseCompilationVMs: true,
	}
}

func (c Compilation) DeleteIfExists(bc remoteclients.BOSHClient) error {
	return bc.DeleteNetwork(c.InternalName())
}

// +kubebuilder:object:root=true

// CompilationList contains a list of Compilation
type CompilationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Compilation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Compilation{}, &CompilationList{})
}
