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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-community/go-uaa"
)

type UAAClient interface {
	HasClient(string) (bool, error)
	CreateClient(string, string, []string) error
	DeleteClient(string) error
}

type uaaClientImpl struct {
	api *uaa.API
}

func NewUAAClient(
	url string,
	clientName string,
	clientSecret string,
	caCert string,
) (UAAClient, error) {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	rootCAs.AppendCertsFromPEM([]byte(caCert))

	targetURL, err := uaa.BuildTargetURL(url)
	if err != nil {
		return nil, err
	}

	api := (&uaa.API{
		UserAgent: "go-uaa",
		TargetURL: targetURL,
	}).WithClient(
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: rootCAs},
			},
		},
	).WithClientCredentials(
		clientName,
		clientSecret,
		uaa.JSONWebToken,
	)

	if err = api.Validate(); err != nil {
		return nil, err
	} else {
		return &uaaClientImpl{api: api}, nil
	}
}

func (c *uaaClientImpl) HasClient(name string) (bool, error) {
	if clients, _, err := c.api.ListClients(
		fmt.Sprintf("client_id eq \"%s\"", name),
		"client_id",
		"ascending",
		1,
		2,
	); err != nil {
		return false, err
	} else {
		return len(clients) == 1, nil
	}
}

func (c *uaaClientImpl) CreateClient(name, secret string, authorities []string) error {
	_, err := c.api.CreateClient(
		uaa.Client{
			ClientID:             name,
			ClientSecret:         secret,
			AuthorizedGrantTypes: []string{"client_credentials"},
			Scope:                []string{"uaa.none"},
			Authorities:          authorities,
		},
	)
	return err
}

func (c *uaaClientImpl) DeleteClient(name string) error {
	_, err := c.api.DeleteClient(name)
	return err
}
