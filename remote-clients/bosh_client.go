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
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type BOSHClient interface {
	HasRelease(string, string) (bool, error)
	UploadRelease(string, string) error
	DeleteRelease(string, string) error

	HasStemcell(string, string) (bool, error)
	UploadStemcell(string, string) error
	DeleteStemcell(string, string) error
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
