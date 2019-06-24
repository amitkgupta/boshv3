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
)

// StemcellReconciler reconciles a Stemcell object
type StemcellReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=stemcells,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=stemcells/status,verbs=get;update;patch

func (r *StemcellReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("stemcell", req.NamespacedName)

	var stemcell boshv1.Stemcell
	if err := r.Get(ctx, req.NamespacedName, &stemcell); err != nil {
		log.Error(err, "unable to fetch stemcell")
		return ctrl.Result{}, ignoreDoesNotExist(err)
	}

	if bc, err := boshClientForNamespace(
		ctx,
		r.Client,
		req.NamespacedName.Namespace,
	); err != nil {
		return ctrl.Result{}, err
	} else {
		return ctrl.Result{}, reconcileWithBOSH(
			ctx,
			r.Client,
			bc,
			&stemcell,
		)
	}
}

func (r *StemcellReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&boshv1.Stemcell{}).
		Complete(r)
}