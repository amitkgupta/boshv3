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

package controllers

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	boshv1 "github.com/amitkgupta/boshv3/api/v1"
	"github.com/amitkgupta/boshv3/remote-clients"
)

type extensional interface {
	BeingDeleted() bool
	EnsureNoFinalizer() bool
	EnsureFinalizer() bool
}

type boshArtifact interface {
	runtime.Object
	extensional
	PrepareToSave() bool
	CreateUnlessExists(remoteclients.BOSHClient) error
	DeleteIfExists(remoteclients.BOSHClient) error
}

type uaaEntity interface {
	runtime.Object
	extensional
	PrepareToSave(string) bool
	SecretName() string
	SecretNamespace() string
	CreateUnlessExists(remoteclients.UAAClient, string) error
	DeleteIfExists(remoteclients.UAAClient) error
}

func reconcileWithBOSH(ctx context.Context, c client.Client, bc remoteclients.BOSHClient, ba boshArtifact) error {
	if ba.BeingDeleted() {
		if err := ba.DeleteIfExists(bc); err != nil {
			return err
		}

		if ba.EnsureNoFinalizer() {
			if err := c.Update(ctx, ba); err != nil {
				return err
			}
		}

		return nil
	}

	if ba.PrepareToSave() {
		if err := c.Status().Update(ctx, ba); err != nil {
			return err
		}
	}

	if ba.EnsureFinalizer() {
		if err := c.Update(ctx, ba); err != nil {
			return err
		}
	}

	if err := ba.CreateUnlessExists(bc); err != nil {
		return err
	} else {
		return c.Status().Update(ctx, ba)
	}
}

func boshClientForNamespace(
	ctx context.Context,
	c client.Client,
	boshSystemNamespace string,
	namespace string,
) (remoteclients.BOSHClient, error) {
	var teams boshv1.TeamList
	if err := c.List(ctx, &teams, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	if len(teams.Items) == 0 {
		return nil, errors.New(fmt.Sprintf("No team assigned to '%s' namespace", namespace))
	}

	if len(teams.Items) > 1 {
		return nil, errors.New(fmt.Sprintf("Detected %d teams in '%s' namespace", len(teams.Items), namespace))
	}

	team := teams.Items[0]

	var secret v1.Secret
	if err := c.Get(
		ctx,
		types.NamespacedName{
			Namespace: team.SecretNamespace(),
			Name:      team.SecretName(),
		},
		&secret,
	); err != nil {
		return nil, err
	}

	var director boshv1.Director
	if err := c.Get(
		ctx,
		types.NamespacedName{
			Namespace: boshSystemNamespace,
			Name:      team.Status.OriginalDirector,
		},
		&director,
	); err != nil {
		return nil, err
	}

	return remoteclients.NewBOSHClient(
		director.Spec.URL,
		director.Spec.CACert,
		director.Spec.UAAURL,
		team.ClientName(),
		string(secret.Data["secret"]),
		director.Spec.UAACACert,
	)
}
