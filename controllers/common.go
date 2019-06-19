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

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
)

type boshArtifact interface {
	runtime.Object

	BeingDeleted() bool

	HasFinalizer() bool
	EnsureFinalizer() bool
	RemoveFinalizer()

	SaveOriginalSpec() (bool, bool)

	EnsureWarning() bool
	EnsureNoWarning() bool

	EnsureAbsentFromDirector() bool
	EnsurePresentOnDirector() bool

	PresentOnDirector(boshdir.Director) (bool, error)
	UploadToDirector(boshdir.Director) error
	DeleteFromDirector(boshdir.Director) error
}

func reconcileWithDirector(ctx context.Context, c client.Client, d boshdir.Director, ba boshArtifact) error {
	if ba.BeingDeleted() {
		if ba.HasFinalizer() {
			if err := ba.DeleteFromDirector(d); err != nil {
				return err
			}

			ba.RemoveFinalizer()
			if err := c.Update(ctx, ba); err != nil {
				return err
			}
		}

		return nil
	} else {
		if changed := ba.EnsureFinalizer(); changed {
			if err := c.Update(ctx, ba); err != nil {
				return err
			}
		}
	}

	saved, mutated := ba.SaveOriginalSpec()
	if saved {
		if err := c.Status().Update(ctx, ba); err != nil {
			return err
		}
	}

	if mutated {
		if changed := ba.EnsureWarning(); changed {
			if err := c.Status().Update(ctx, ba); err != nil {
				return err
			}
		}
	} else {
		if changed := ba.EnsureNoWarning(); changed {
			if err := c.Status().Update(ctx, ba); err != nil {
				return err
			}
		}
	}

	if present, err := ba.PresentOnDirector(d); err != nil {
		return err
	} else if !present {
		if changed := ba.EnsureAbsentFromDirector(); changed {
			if err := c.Status().Update(ctx, ba); err != nil {
				return err
			}
		}

		if err := ba.UploadToDirector(d); err != nil {
			return err
		}

		if changed := ba.EnsurePresentOnDirector(); changed {
			if err := c.Status().Update(ctx, ba); err != nil {
				return err
			}
		}
	}

	return nil
}
