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
	"github.com/amitkgupta/boshv3/remote-clients"
)

// CompilationReconciler reconciles a Compilation object
type CompilationReconciler struct {
	client.Client
	Log                 logr.Logger
	BOSHSystemNamespace string
}

// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=compilations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=compilations/status,verbs=get;update;patch

func (r *CompilationReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, err error) {
	ctx := context.Background()
	log := r.Log.WithValues("compilation", req.NamespacedName)

	if req.NamespacedName.Namespace != r.BOSHSystemNamespace {
		msg := "cannot create compilation outside BOSH system namespace"
		err = errors.New(msg)
		log.Error(
			err,
			msg,
			"namespace", req.NamespacedName.Namespace,
			"bosh_system_namespace", r.BOSHSystemNamespace,
		)
		return
	}

	var compilation boshv1.Compilation
	if err = r.Get(ctx, req.NamespacedName, &compilation); err != nil {
		log.Error(err, "unable to fetch compilation")
		err = ignoreDoesNotExist(err)
		return
	}

	if compilation.PrepareToSave() {
		if err = r.Status().Update(ctx, &compilation); err != nil {
			log.Error(err, "unable to save compilation")
			return
		}
	}

	var bc remoteclients.BOSHClient
	if bc, err = boshClientForDirector(
		ctx,
		log,
		r.Client,
		r.BOSHSystemNamespace,
		compilation.Status.OriginalDirector,
	); err != nil {
		log.Error(
			err,
			"unable to construct BOSH client for director",
			"director", compilation.Status.OriginalDirector,
		)
		return
	}

	if err = reconcileWithBOSH(ctx, log, r.Client, bc, &compilation); err != nil {
		log.Error(err, "unable to reconcile with BOSH")
		return
	}

	return
}

func (r *CompilationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&boshv1.Compilation{}).
		Complete(r)
}
