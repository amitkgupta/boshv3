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
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/amitkgupta/boshv3/remote-clients"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NetworkSpec defines the desired state of Network
type NetworkSpec struct {
	Type    string   `json:"type"`
	Subnets []Subnet `json:"subnets"`
}

func (n1 NetworkSpec) match(n2 NetworkSpec) bool {
	if n1.Type != n2.Type {
		return false
	}

	if len(n1.Subnets) != len(n2.Subnets) {
		return false
	}

	for i, s1 := range n1.Subnets {
		if !s1.match(n2.Subnets[i]) {
			return false
		}
	}

	return true
}

type Subnet struct {
	Range           string                `json:"range"`
	Gateway         string                `json:"gateway"`
	DNS             []string              `json:"dns"`
	Reserved        []string              `json:"reserved"`
	Static          []string              `json:"static"`
	AZs             []string              `json:"azs"`
	CloudProperties *runtime.RawExtension `json:"cloud_properties,omitempty"`
}

func (s1 Subnet) match(s2 Subnet) bool {
	if s1.Range != s2.Range {
		return false
	}

	if s1.Gateway != s2.Gateway {
		return false
	}

	if len(s1.DNS) != len(s2.DNS) {
		return false
	}

	sortedDNS1 := sort.StringSlice(s1.DNS)
	sortedDNS1.Sort()

	sortedDNS2 := sort.StringSlice(s2.DNS)
	sortedDNS2.Sort()

	for i, d1 := range sortedDNS1 {
		if d1 != sortedDNS2[i] {
			return false
		}
	}

	if len(s1.Reserved) != len(s2.Reserved) {
		return false
	}

	sortedReserved1 := sort.StringSlice(s1.Reserved)
	sortedReserved1.Sort()

	sortedReserved2 := sort.StringSlice(s2.Reserved)
	sortedReserved2.Sort()

	for i, r1 := range sortedReserved1 {
		if r1 != sortedReserved2[i] {
			return false
		}
	}

	if len(s1.Static) != len(s2.Static) {
		return false
	}

	sortedStatic1 := sort.StringSlice(s1.Static)
	sortedStatic1.Sort()

	sortedStatic2 := sort.StringSlice(s2.Static)
	sortedStatic2.Sort()

	for i, s1 := range sortedStatic1 {
		if s1 != sortedStatic2[i] {
			return false
		}
	}

	if len(s1.AZs) != len(s2.AZs) {
		return false
	}

	sortedAZs1 := sort.StringSlice(s1.AZs)
	sortedAZs1.Sort()

	sortedAZs2 := sort.StringSlice(s2.AZs)
	sortedAZs2.Sort()

	for i, a1 := range sortedAZs1 {
		if a1 != sortedAZs2[i] {
			return false
		}
	}

	if s1.CloudProperties.String() != s2.CloudProperties.String() {
		return false
	}

	return true
}

// NetworkStatus defines the observed state of Network
type NetworkStatus struct {
	Warning      string      `json:"warning"`
	OriginalSpec NetworkSpec `json:"original_spec"`
	Available    bool        `json:"available"`
}

// +kubebuilder:object:root=true

// Network is the Schema for the networks API
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkSpec   `json:"spec,omitempty"`
	Status NetworkStatus `json:"status,omitempty"`
}

func (n Network) BeingDeleted() bool {
	return !n.GetDeletionTimestamp().IsZero()
}

var networkFinalizer = strings.Join([]string{"network", finalizerBase}, ".")

func (n Network) hasFinalizer() bool {
	return containsString(n.GetFinalizers(), networkFinalizer)
}

func (n *Network) EnsureFinalizer() bool {
	changed := !n.hasFinalizer()
	n.SetFinalizers(append(n.GetFinalizers(), networkFinalizer))
	return changed
}

func (n *Network) EnsureNoFinalizer() bool {
	changed := n.hasFinalizer()
	n.SetFinalizers(removeString(n.GetFinalizers(), networkFinalizer))
	return changed
}

func (n *Network) PrepareToSave() (needsStatusUpdate bool) {
	originalSpec := n.Status.OriginalSpec

	if originalSpec.Type == "" {
		n.Status.OriginalSpec = n.Spec
		needsStatusUpdate = true
	} else {
		mutated := !originalSpec.match(n.Spec)

		if mutated && n.Status.Warning == "" {
			n.Status.Warning = resourceMutationWarning
			needsStatusUpdate = true
		} else if !mutated && n.Status.Warning != "" {
			n.Status.Warning = ""
			needsStatusUpdate = true
		}
	}

	return
}

func (n Network) InternalName() string {
	return strings.Join([]string{
		"network",
		n.GetNamespace(),
		n.GetName(),
	}, "-")
}

func (n *Network) CreateUnlessExists(
	bc remoteclients.BOSHClient,
	ctx context.Context,
	c client.Client,
) error {
	network, err := n.resolveReferences(ctx, c)
	if err != nil {
		return err
	}

	if err := bc.CreateNetwork(n.InternalName(), network); err != nil {
		return err
	}

	n.Status.Available = true

	return nil
}

func (n Network) resolveReferences(ctx context.Context, c client.Client) (remoteclients.Network, error) {
	spec := n.Status.OriginalSpec
	network := remoteclients.Network{
		Name:    n.InternalName(),
		Type:    spec.Type,
		Subnets: make([]remoteclients.Subnet, len(spec.Subnets)),
	}

	for i, s := range spec.Subnets {
		network.Subnets[i] = remoteclients.Subnet{
			Range:           s.Range,
			Gateway:         s.Gateway,
			DNS:             s.DNS,
			Reserved:        s.Reserved,
			Static:          s.Static,
			CloudProperties: s.CloudProperties,
			AZs:             make([]string, len(s.AZs)),
		}

		for j, a := range s.AZs {
			var az AZ
			if err := c.Get(
				ctx,
				types.NamespacedName{
					Namespace: n.GetNamespace(),
					Name:      a,
				},
				&az,
			); err != nil {
				return remoteclients.Network{}, err
			}

			network.Subnets[i].AZs[j] = az.InternalName()
		}
	}

	return network, nil
}

func (n Network) DeleteIfExists(bc remoteclients.BOSHClient) error {
	return bc.DeleteNetwork(n.InternalName())
}

// +kubebuilder:object:root=true

// NetworkList contains a list of Network
type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Network `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Network{}, &NetworkList{})
}
