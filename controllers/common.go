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

	"github.com/go-logr/logr"

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
	CreateUnlessExists(remoteclients.BOSHClient, context.Context, client.Client) error
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

func boshClientForNamespace(
	ctx context.Context,
	log logr.Logger,
	c client.Client,
	boshSystemNamespace string,
	namespace string,
) (remoteclients.BOSHClient, error) {
	log = log.WithValues("namespace", namespace, "bosh_system_namespace", boshSystemNamespace)

	var teams boshv1.TeamList
	if err := c.List(ctx, &teams, client.InNamespace(namespace)); err != nil {
		log.Error(err, "failed to list teams")
		return nil, err
	}

	if len(teams.Items) == 0 {
		msg := "No team assigned to namespace"
		err := errors.New(msg)
		log.Error(err, msg)
		return nil, err
	}

	if len(teams.Items) > 1 {
		msg := fmt.Sprintf("Found %d teams in namespace", len(teams.Items))
		err := errors.New(msg)
		log.Error(err, msg)
		return nil, err
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
		log.Error(
			err,
			"failed to get secret",
			"secret", team.SecretName(),
			"secret_namespace", team.SecretNamespace(),
		)
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
		log.Error(err, "failed to get director")
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

func reconcileWithBOSH(
	ctx context.Context,
	log logr.Logger,
	c client.Client,
	bc remoteclients.BOSHClient,
	ba boshArtifact,
) error {
	if ba.BeingDeleted() {
		if err := ba.DeleteIfExists(bc); err != nil {
			log.Error(err, "failed to delete if exists in BOSH")
			return err
		}

		if ba.EnsureNoFinalizer() {
			if err := c.Update(ctx, ba); err != nil {
				log.Error(err, "failed to updated after ensuring no finalizer")
				return err
			}
		}

		return nil
	}

	if ba.PrepareToSave() {
		if err := c.Status().Update(ctx, ba); err != nil {
			log.Error(err, "failed to updated after preparing to save")
			return err
		}
	}

	if ba.EnsureFinalizer() {
		if err := c.Update(ctx, ba); err != nil {
			log.Error(err, "failed to update after ensuring finalizer")
			return err
		}
	}

	if err := ba.CreateUnlessExists(bc, ctx, c); err != nil {
		log.Error(err, "failed to create unless exists in BOSH")
		return err
	}

	if err := c.Status().Update(ctx, ba); err != nil {
		log.Error(err, "failed to update after creating unless exits in BOSH")
		return err
	}

	return nil
}
