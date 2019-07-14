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
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/amitkgupta/boshv3/remote-clients"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeploymentSpec defines the desired state of Deployment
type DeploymentSpec struct {
	AZs                 []string       `json:"azs"`
	Replicas            int            `json:"replicas"`
	Containers          []Container    `json:"containers"`
	Extensions          []string       `json:"extensions,omitempty"`
	BaseImage           string         `json:"base_image"`
	Network             string         `json:"network"`
	UpdateStrategy      UpdateStrategy `json:"update_strategy"`
	ForceReconciliation bool           `json:"force_reconciliation"`
}

type Container struct {
	Role                  string                           `json:"role"`
	ExportedConfiguration map[string]ExportedConfiguration `json:"exported_configuration,omitempty"`
	ImportedConfiguration map[string]ImportedConfiguration `json:"imported_configuration,omitempty"`
	Resources             Resources                        `json:"resources"`
}

type Resources struct {
	RAM                int `json:"ram"`
	CPU                int `json:"cpu"`
	EphemeralDiskSize  int `json:"ephemeral_disk_size"`
	PersistentDiskSize int `json:"persistent_disk_size,omitempty"`
}

type ExportedConfiguration struct {
	InternalLink string `json:"internal_link"`
	Exported     bool   `json:"exported,omitempty"`
}

type ImportedConfiguration struct {
	InternalLink string `json:"internal_link"`
	ImportedFrom string `json:"imported_from,omitempty"`
}

type UpdateStrategy struct {
	MinReadySeconds        int    `json:"min_ready_seconds,omitempty"`
	MaxReadySeconds        int    `json:"max_ready_seconds,omitempty"`
	MaxUnavailablePercent  string `json:"max_unavailable_percent,omitempty"`
	MaxUnavailableReplicas int    `json:"max_unavailable_replicas,omitempty"`
	Type                   string `json:"type,omitempty"`
}

// DeploymentStatus defines the observed state of Deployment
type DeploymentStatus struct {
	Available bool `json:"available"`
}

// +kubebuilder:object:root=true

// Deployment is the Schema for the deployments API
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentSpec   `json:"spec,omitempty"`
	Status DeploymentStatus `json:"status,omitempty"`
}

func (d Deployment) BeingDeleted() bool {
	return !d.GetDeletionTimestamp().IsZero()
}

var deploymentFinalizer = strings.Join([]string{"deployment", finalizerBase}, ".")

func (d Deployment) hasFinalizer() bool {
	return containsString(d.GetFinalizers(), deploymentFinalizer)
}

func (d *Deployment) EnsureFinalizer() bool {
	changed := !d.hasFinalizer()
	d.SetFinalizers(append(d.GetFinalizers(), deploymentFinalizer))
	return changed
}

func (d *Deployment) EnsureNoFinalizer() bool {
	changed := d.hasFinalizer()
	d.SetFinalizers(removeString(d.GetFinalizers(), deploymentFinalizer))
	return changed
}

func (d Deployment) PrepareToSave() bool {
	return false
}

func (d Deployment) InternalName() string {
	return strings.Join([]string{
		"deployment",
		d.GetNamespace(),
		d.GetName(),
	}, "-")
}

func (d *Deployment) CreateUnlessExists(
	bc remoteclients.BOSHClient,
	ctx context.Context,
	c client.Client,
) error {
	deployment, err := d.resolveReferences(ctx, c)
	if err != nil {
		return err
	}

	if err := bc.CreateDeployment(d.InternalName(), deployment); err != nil {
		return err
	}

	d.Status.Available = true

	return nil
}

func (d *Deployment) resolveReferences(ctx context.Context, c client.Client) (remoteclients.Deployment, error) {
	deployment := remoteclients.Deployment{
		Name: d.InternalName(),
		Update: remoteclients.DeploymentUpdate{
			Canaries:        0,
			MaxInFlight:     d.maxUnavailable(),
			CanaryWatchTime: d.watchTime(),
			UpdateWatchTime: d.watchTime(),
			Serial:          false,
			VMStrategy:      d.vmStrategy(),
		},
	}

	if releases, err := d.releases(ctx, c); err != nil {
		return remoteclients.Deployment{}, err
	} else {
		deployment.Releases = releases
	}

	if stemcell, err := d.stemcell(ctx, c); err != nil {
		return remoteclients.Deployment{}, err
	} else {
		deployment.Stemcells = []remoteclients.Stemcell{stemcell}
	}

	if instanceGroup, err := d.instanceGroup(ctx, c); err != nil {
		return remoteclients.Deployment{}, err
	} else {
		deployment.InstanceGroups = []remoteclients.InstanceGroup{instanceGroup}
	}

	return deployment, nil
}

func (d Deployment) maxUnavailable() interface{} {
	if d.Spec.UpdateStrategy.MaxUnavailablePercent == "" {
		return d.Spec.UpdateStrategy.MaxUnavailableReplicas
	} else {
		return d.Spec.UpdateStrategy.MaxUnavailablePercent
	}
}

func (d Deployment) watchTime() interface{} {
	if d.Spec.UpdateStrategy.MaxReadySeconds == 0 {
		return 1000 * d.Spec.UpdateStrategy.MinReadySeconds
	} else {
		return fmt.Sprintf(
			"%d-%d",
			1000*d.Spec.UpdateStrategy.MinReadySeconds,
			1000*d.Spec.UpdateStrategy.MaxReadySeconds,
		)
	}
}

func (d Deployment) vmStrategy() string {
	if d.Spec.UpdateStrategy.Type == "" {
		return "delete-create"
	} else {
		return d.Spec.UpdateStrategy.Type
	}
}

func (d Deployment) releases(ctx context.Context, c client.Client) ([]remoteclients.Release, error) {
	uniqueReleases := make(map[remoteclients.Release]struct{})

	for _, container := range d.Spec.Containers {
		var role Role
		if err := c.Get(
			ctx,
			types.NamespacedName{
				Namespace: d.GetNamespace(),
				Name:      container.Role,
			},
			&role,
		); err != nil {
			return nil, err
		}

		var release Release
		if err := c.Get(
			ctx,
			types.NamespacedName{
				Namespace: d.GetNamespace(),
				Name:      role.Spec.Source.Release,
			},
			&release,
		); err != nil {
			return nil, err
		}

		uniqueReleases[remoteclients.Release{
			Name:    release.Status.OriginalSpec.ReleaseName,
			Version: release.Status.OriginalSpec.Version,
		}] = struct{}{}
	}

	releases := []remoteclients.Release{}
	for r, _ := range uniqueReleases {
		releases = append(releases, r)
	}
	return releases, nil
}

const stemcellAlias = "stemcell"

func (d Deployment) stemcell(ctx context.Context, c client.Client) (remoteclients.Stemcell, error) {
	var baseImage BaseImage
	if err := c.Get(
		ctx,
		types.NamespacedName{
			Namespace: d.GetNamespace(),
			Name:      d.Spec.BaseImage,
		},
		&baseImage,
	); err != nil {
		return remoteclients.Stemcell{}, err
	}

	return remoteclients.Stemcell{
		Alias:   stemcellAlias,
		Name:    baseImage.Status.OriginalSpec.BaseImageName,
		Version: baseImage.Status.OriginalSpec.Version,
	}, nil
}

func (d Deployment) instanceGroup(ctx context.Context, c client.Client) (remoteclients.InstanceGroup, error) {
	instanceGroup := remoteclients.InstanceGroup{
		Name:         d.InternalName(),
		AZs:          make([]string, len(d.Spec.AZs)),
		Instances:    d.Spec.Replicas,
		Jobs:         make([]remoteclients.Job, len(d.Spec.Containers)),
		VMExtensions: make([]string, len(d.Spec.Extensions)),
		VMResources: remoteclients.VMResources{
			RAM:               d.ram(),
			CPU:               d.cpu(),
			EphemeralDiskSize: d.ephemeralDiskSize(),
		},
		Stemcell:           stemcellAlias,
		PersistentDiskSize: d.persistentDiskSize(),
	}

	for i, azName := range d.Spec.AZs {
		var az AZ
		if err := c.Get(
			ctx,
			types.NamespacedName{
				Namespace: d.GetNamespace(),
				Name:      azName,
			},
			&az,
		); err != nil {
			return remoteclients.InstanceGroup{}, err
		}

		instanceGroup.AZs[i] = az.InternalName()
	}

	for i, container := range d.Spec.Containers {
		var role Role
		if err := c.Get(
			ctx,
			types.NamespacedName{
				Namespace: d.GetNamespace(),
				Name:      container.Role,
			},
			&role,
		); err != nil {
			return remoteclients.InstanceGroup{}, err
		}

		var release Release
		if err := c.Get(
			ctx,
			types.NamespacedName{
				Namespace: d.GetNamespace(),
				Name:      role.Spec.Source.Release,
			},
			&release,
		); err != nil {
			return remoteclients.InstanceGroup{}, err
		}

		instanceGroup.Jobs[i] = remoteclients.Job{
			Name:       role.Spec.Source.Job,
			Release:    release.Status.OriginalSpec.ReleaseName,
			Consumes:   make(map[string]remoteclients.ConsumesLink),
			Provides:   make(map[string]remoteclients.ProvidesLink),
			Properties: role.Spec.Properties,
		}

		for externalLink, configuration := range container.ImportedConfiguration {
			var d2 Deployment
			if err := c.Get(
				ctx,
				types.NamespacedName{
					Namespace: d.GetNamespace(),
					Name:      configuration.ImportedFrom,
				},
				&d2,
			); err != nil {
				return remoteclients.InstanceGroup{}, err
			}

			instanceGroup.Jobs[i].Consumes[configuration.InternalLink] = remoteclients.ConsumesLink{
				From:       externalLink,
				Deployment: d2.InternalName(),
			}
		}

		for externalLink, configuration := range container.ExportedConfiguration {
			instanceGroup.Jobs[i].Provides[configuration.InternalLink] = remoteclients.ProvidesLink{
				As:     externalLink,
				Shared: configuration.Exported,
			}
		}
	}

	for i, extensionName := range d.Spec.Extensions {
		var extension VMExtension
		if err := c.Get(
			ctx,
			types.NamespacedName{
				Namespace: d.GetNamespace(),
				Name:      extensionName,
			},
			&extension,
		); err != nil {
			return remoteclients.InstanceGroup{}, err
		}

		instanceGroup.VMExtensions[i] = extension.InternalName()
	}

	var network Network
	if err := c.Get(
		ctx,
		types.NamespacedName{
			Namespace: d.GetNamespace(),
			Name:      d.Spec.Network,
		},
		&network,
	); err != nil {
		return remoteclients.InstanceGroup{}, err
	} else {
		instanceGroup.Networks = []remoteclients.DeploymentNetwork{
			remoteclients.DeploymentNetwork{Name: network.InternalName()},
		}
	}

	return instanceGroup, nil
}

func (d Deployment) ram() int {
	ram := 0
	for _, c := range d.Spec.Containers {
		ram += c.Resources.RAM
	}
	return ram
}

func (d Deployment) cpu() int {
	cpu := 0
	for _, c := range d.Spec.Containers {
		cpu += c.Resources.CPU
	}
	return cpu
}

func (d Deployment) ephemeralDiskSize() int {
	ephemeralDiskSize := 0
	for _, c := range d.Spec.Containers {
		ephemeralDiskSize += c.Resources.EphemeralDiskSize
	}
	return ephemeralDiskSize
}

func (d Deployment) persistentDiskSize() int {
	persistentDiskSize := 0
	for _, c := range d.Spec.Containers {
		persistentDiskSize += c.Resources.PersistentDiskSize
	}
	return persistentDiskSize
}

func (d Deployment) DeleteIfExists(bc remoteclients.BOSHClient) error {
	return bc.DeleteDeployment(d.InternalName())
}

// +kubebuilder:object:root=true

// DeploymentList contains a list of Deployment
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Deployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Deployment{}, &DeploymentList{})
}
