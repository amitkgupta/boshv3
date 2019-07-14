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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	boshv1 "github.com/amitkgupta/boshv3/api/v1"
	"github.com/amitkgupta/boshv3/remote-clients"
)

// ExtensionReconciler reconciles a Extension object
type ExtensionReconciler struct {
	client.Client
	Log                 logr.Logger
	BOSHSystemNamespace string
}

// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=extensions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=extensions/status,verbs=get;update;patch

func (r *ExtensionReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, err error) {
	ctx := context.Background()
	log := r.Log.WithValues("extension", req.NamespacedName)

	var extension boshv1.Extension
	if err = r.Get(ctx, req.NamespacedName, &extension); err != nil {
		log.Error(err, "unable to fetch extension")
		err = ignoreDoesNotExist(err)
		return
	}

	var bc remoteclients.BOSHClient
	if bc, err = boshClientForNamespace(
		ctx,
		log,
		r.Client,
		r.BOSHSystemNamespace,
		req.NamespacedName.Namespace,
	); err != nil {
		log.Error(err, "unable to construct BOSH client for namespace", "namespace", req.NamespacedName.Namespace)
		return
	}

	if err = reconcileWithBOSH(ctx, log, r.Client, bc, &extension); err != nil {
		log.Error(err, "unable to reconcile with BOSH")
		return
	}

	return
}

func (r *ExtensionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&boshv1.Extension{}).
		Complete(r)
}
