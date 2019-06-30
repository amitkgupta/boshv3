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

	"github.com/cloudfoundry/bosh-cli/cmd/config/configfakes"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	"k8s.io/apimachinery/pkg/runtime"
)

type BOSHClient interface {
	HasRelease(string, string) (bool, error)
	UploadRelease(string, string) error
	DeleteRelease(string, string) error

	HasStemcell(string, string) (bool, error)
	UploadStemcell(string, string) error
	DeleteStemcell(string, string) error

	CreateVMExtension(string, json.Marshaler) error
	DeleteVMExtension(string) error

	CreateAZ(string, json.Marshaler) error
	DeleteAZ(string) error

	CreateNetwork(string, Network) error
	DeleteNetwork(string) error
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
		new(configfakes.FakeConfig),
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

func (c *boshClientImpl) HasStemcell(stemcellName, version string) (bool, error) {
	return c.api.HasStemcell(stemcellName, version)
}

func (c *boshClientImpl) UploadStemcell(url, sha1 string) error {
	return c.api.UploadStemcellURL(url, sha1, false)
}

func (c *boshClientImpl) DeleteStemcell(stemcellName, version string) error {
	if stemcell, err := c.api.FindStemcell(boshdir.NewStemcellSlug(
		stemcellName,
		version,
	)); err != nil {
		return err
	} else {
		return stemcell.Delete(false)
	}
}

type az struct {
	Name            string         `json:"name"`
	CloudProperties json.Marshaler `json:"cloud_properties"`
}

type vmExtension struct {
	Name            string         `json:"name"`
	CloudProperties json.Marshaler `json:"cloud_properties"`
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
	Reserved        []string              `json:"reserved"`
	Static          []string              `json:"static"`
	AZs             []string              `json:"azs"`
	CloudProperties *runtime.RawExtension `json:"cloud_properties,omitempty"`
}

type cloudConfig struct {
	AZs          []az          `json:"azs,omitempty"`
	VMExtensions []vmExtension `json:"vm_extensions,omitempty"`
	Networks     []Network     `json:"networks,omitempty"`
}

func (c *boshClientImpl) CreateVMExtension(name string, cloudProperties json.Marshaler) error {
	return c.updateCloudConfig(
		name,
		cloudConfig{
			VMExtensions: []vmExtension{vmExtension{
				Name:            name,
				CloudProperties: cloudProperties,
			}},
		},
	)
}

func (c *boshClientImpl) DeleteVMExtension(name string) error {
	return c.deleteCloudConfig(name)
}

func (c *boshClientImpl) CreateAZ(name string, cloudProperties json.Marshaler) error {
	return c.updateCloudConfig(
		name,
		cloudConfig{
			AZs: []az{az{
				Name:            name,
				CloudProperties: cloudProperties,
			}},
		},
	)
}

func (c *boshClientImpl) DeleteAZ(name string) error {
	return c.deleteCloudConfig(name)
}

func (c *boshClientImpl) CreateNetwork(name string, network Network) error {
	return c.updateCloudConfig(
		name,
		cloudConfig{
			Networks: []Network{network},
		},
	)
}

func (c *boshClientImpl) DeleteNetwork(name string) error {
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
