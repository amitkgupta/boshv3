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

	"github.com/go-logr/logr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	boshv1 "github.com/amitkgupta/boshv3/api/v1"
)

// DirectorReconciler reconciles a Director object
type DirectorReconciler struct {
	client.Client
	Log                 logr.Logger
	BOSHSystemNamespace string
}

// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=directors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=directors/status,verbs=get;update;patch

func (r *DirectorReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, err error) {
	ctx := context.Background()
	log := r.Log.WithValues("director", req.NamespacedName)

	if req.NamespacedName.Namespace != r.BOSHSystemNamespace {
		msg := "cannot create director outside BOSH system namespace"
		err = errors.New(msg)
		log.Error(
			err,
			msg,
			"namespace", req.NamespacedName.Namespace,
			"bosh_system_namespace", r.BOSHSystemNamespace,
		)
		return
	}

	director := new(boshv1.Director)
	if err = r.Get(ctx, req.NamespacedName, director); err != nil {
		log.Error(err, "unable to fetch director")
		err = ignoreDoesNotExist(err)
		return
	}

	team := director.Team()

	if director.BeingDeleted() {
		if err = ignoreDoesNotExist(r.Delete(ctx, &team)); err != nil {
			log.Error(err, "failed to delete team", "team", team.GetName())
			return
		}

		if director.EnsureNoFinalizer() {
			if err = r.Update(ctx, director); err != nil {
				log.Error(err, "failed to update after ensuring no finalizer")
				return
			}
		}

		return
	}

	if director.EnsureFinalizer() {
		if err = r.Update(ctx, director); err != nil {
			log.Error(err, "failed to update after ensuring finalizer")
			return
		}
	}

	if err = ignoreAlreadyExists(r.Create(ctx, &team)); err != nil {
		log.Error(err, "failed to create director team", "team", team.GetName())
		return
	}

	return
}

func (r *DirectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&boshv1.Director{}).
		Complete(r)
}
