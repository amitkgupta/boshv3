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

package remoteclients

import (
	"encoding/json"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	"k8s.io/apimachinery/pkg/runtime"
)

type BOSHClient interface {
	HasRelease(string, string) (bool, error)
	UploadRelease(string, string) error
	DeleteRelease(string, string) error

	HasBaseImage(string, string) (bool, error)
	UploadBaseImage(string, string) error
	DeleteBaseImage(string, string) error

	CreateVMExtension(string, VMExtension) error
	DeleteVMExtension(string) error

	CreateAZ(string, AZ) error
	DeleteAZ(string) error

	CreateNetwork(string, Network) error
	DeleteNetwork(string) error

	CreateCompilation(string, Network, AZ, Compilation) error
	DeleteCompilation(string) error

	CreateDeployment(string, Deployment) error
	DeleteDeployment(string) error
}

type boshClientImpl struct {
	api boshdir.Director
}

func NewBOSHClient(
	url string,
	caCert string,
	uaaURL string,
	uaaClientName string,
	uaaClientSecret string,
	uaaCACert string,
) (BOSHClient, error) {
	logger := boshlog.NewLogger(boshlog.LevelError)

	uaaConfig, err := boshuaa.NewConfigFromURL(uaaURL)
	if err != nil {
		return nil, err
	}

	uaaConfig.Client = uaaClientName
	uaaConfig.ClientSecret = uaaClientSecret
	uaaConfig.CACert = uaaCACert

	uaa, err := boshuaa.NewFactory(logger).New(uaaConfig)
	if err != nil {
		return nil, err
	}

	directorConfig, err := boshdir.NewConfigFromURL(url)
	if err != nil {
		return nil, err
	}

	directorConfig.CACert = caCert
	directorConfig.TokenFunc = boshuaa.NewClientTokenSession(uaa).TokenFunc

	api, err := boshdir.NewFactory(logger).New(
		directorConfig,
		boshdir.NewNoopTaskReporter(),
		boshdir.NewNoopFileReporter(),
	)
	if err != nil {
		return nil, err
	}

	return &boshClientImpl{api: api}, nil
}

func (c *boshClientImpl) HasRelease(releaseName, version string) (bool, error) {
	return c.api.HasRelease(releaseName, version, boshdir.OSVersionSlug{})
}

func (c *boshClientImpl) UploadRelease(url, sha1 string) error {
	return c.api.UploadReleaseURL(url, sha1, false, false)
}

func (c *boshClientImpl) DeleteRelease(releaseName, version string) error {
	if release, err := c.api.FindRelease(boshdir.NewReleaseSlug(
		releaseName,
		version,
	)); err != nil {
		return err
	} else {
		return release.Delete(false)
	}
}

func (c *boshClientImpl) HasBaseImage(baseImageName, version string) (bool, error) {
	return c.api.HasStemcell(baseImageName, version)
}

func (c *boshClientImpl) UploadBaseImage(url, sha1 string) error {
	return c.api.UploadStemcellURL(url, sha1, false)
}

func (c *boshClientImpl) DeleteBaseImage(baseImageName, version string) error {
	if stemcell, err := c.api.FindStemcell(boshdir.NewStemcellSlug(
		baseImageName,
		version,
	)); err != nil {
		return err
	} else {
		return stemcell.Delete(false)
	}
}

type AZ struct {
	Name            string                `json:"name"`
	CloudProperties *runtime.RawExtension `json:"cloud_properties,omitempty"`
}

type VMExtension struct {
	Name            string                `json:"name"`
	CloudProperties *runtime.RawExtension `json:"cloud_properties"`
}

type Network struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Subnets []Subnet `json:"subnets"`
}

type Subnet struct {
	Range           string                `json:"range"`
	Gateway         string                `json:"gateway"`
	DNS             []string              `json:"dns"`
	Reserved        []string              `json:"reserved,omitempty"`
	Static          []string              `json:"static,omitempty"`
	AZs             []string              `json:"azs"`
	CloudProperties *runtime.RawExtension `json:"cloud_properties,omitempty"`
}

type Compilation struct {
	Workers             int                   `json:"workers"`
	AZ                  string                `json:"az"`
	OrphanWorkers       bool                  `json:"orphan_workers"`
	VMResources         VMResources           `json:"vm_resources"`
	CloudProperties     *runtime.RawExtension `json:"cloud_properties,omitempty"`
	Network             string                `json:"network"`
	ReuseCompilationVMs bool                  `json:"reuse_compilation_vms"`
}

type VMResources struct {
	CPU               int `json:"cpu"`
	RAM               int `json:"ram"`
	EphemeralDiskSize int `json:"ephemeral_disk_size"`
}

type cloudConfig struct {
	AZs          []AZ          `json:"azs,omitempty"`
	VMExtensions []VMExtension `json:"vm_extensions,omitempty"`
	Networks     []Network     `json:"networks,omitempty"`
	Compilation  *Compilation  `json:"compilation,omitempty"`
}

type Deployment struct {
	Name           string           `json:"name"`
	Update         DeploymentUpdate `json:"update"`
	Releases       []Release        `json:"releases"`
	Stemcells      []Stemcell       `json:"stemcells"`
	InstanceGroups []InstanceGroup  `json:"instance_groups"`
}

type DeploymentUpdate struct {
	Canaries        int         `json:"canaries"`
	MaxInFlight     interface{} `json:"max_in_flight"`
	CanaryWatchTime interface{} `json:"canary_watch_time"`
	UpdateWatchTime interface{} `json:"update_watch_time"`
	Serial          bool        `json:"serial"`
	VMStrategy      string      `json:"vm_strategy"`
}

type Release struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Stemcell struct {
	Alias   string `json:"alias"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InstanceGroup struct {
	Name               string              `json:"name"`
	AZs                []string            `json:"azs"`
	Instances          int                 `json:"instances"`
	Jobs               []Job               `json:"jobs"`
	VMExtensions       []string            `json:"vm_extensions"`
	VMResources        VMResources         `json:"vm_resources"`
	Stemcell           string              `json:"stemcell"`
	PersistentDiskSize int                 `json:"persistent_disk_size"`
	Networks           []DeploymentNetwork `json:"networks"`
}

type Job struct {
	Name       string                  `json:"name"`
	Release    string                  `json:"release"`
	Consumes   map[string]ConsumesLink `json:"consumes"`
	Provides   map[string]ProvidesLink `json:"provides"`
	Properties *runtime.RawExtension   `json:"properties,omitempty"`
}

type ConsumesLink struct {
	From       string `json:"from"`
	Deployment string `json:"deployment"`
}

type ProvidesLink struct {
	As     string `json:"as"`
	Shared bool   `json:"shared"`
}

type DeploymentNetwork struct {
	Name string `json:"name"`
}

func (c *boshClientImpl) CreateVMExtension(name string, vmExtension VMExtension) error {
	return c.updateCloudConfig(
		name,
		cloudConfig{
			VMExtensions: []VMExtension{vmExtension},
		},
	)
}

func (c *boshClientImpl) DeleteVMExtension(name string) error {
	return c.deleteCloudConfig(name)
}

func (c *boshClientImpl) CreateAZ(name string, az AZ) error {
	return c.updateCloudConfig(name, cloudConfig{AZs: []AZ{az}})
}

func (c *boshClientImpl) DeleteAZ(name string) error {
	return c.deleteCloudConfig(name)
}

func (c *boshClientImpl) CreateNetwork(name string, network Network) error {
	return c.updateCloudConfig(name, cloudConfig{Networks: []Network{network}})
}

func (c *boshClientImpl) DeleteNetwork(name string) error {
	return c.deleteCloudConfig(name)
}

func (c *boshClientImpl) CreateCompilation(
	name string,
	network Network,
	az AZ,
	compilation Compilation,
) error {
	return c.updateCloudConfig(
		name,
		cloudConfig{
			Networks:    []Network{network},
			AZs:         []AZ{az},
			Compilation: &compilation,
		},
	)
}

func (c *boshClientImpl) DeleteCompilation(name string) error {
	return c.deleteCloudConfig(name)
}

func (c *boshClientImpl) updateCloudConfig(
	name string,
	cloudConfig cloudConfig,
) error {
	bytes, err := json.Marshal(cloudConfig)
	if err != nil {
		return err
	}

	configDiff, err := c.api.DiffConfig("cloud", name, bytes)
	if err != nil {
		return err
	}

	_, err = c.api.UpdateConfig("cloud", name, configDiff.FromId, bytes)
	return err
}

func (c *boshClientImpl) deleteCloudConfig(name string) error {
	_, err := c.api.DeleteConfig("cloud", name)
	return err
}

func (c *boshClientImpl) CreateDeployment(name string, deployment Deployment) error {
	bytes, err := json.Marshal(deployment)
	if err != nil {
		return err
	}

	if d, err := c.api.FindDeployment(name); err != nil {
		return err
	} else {
		return d.Update(bytes, boshdir.UpdateOpts{})
	}
}

func (c *boshClientImpl) DeleteDeployment(name string) error {
	if d, err := c.api.FindDeployment(name); err != nil {
		return err
	} else {
		return d.Delete(false)
	}
}
