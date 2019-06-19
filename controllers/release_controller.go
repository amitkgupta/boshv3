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

	boshdir "github.com/cloudfoundry/bosh-cli/director"
)

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Log      logr.Logger
	Director boshdir.Director
}

// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=releases/status,verbs=get;update;patch

func (r *ReleaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("release", req.NamespacedName)

	var release boshv1.Release
	if err := r.Get(ctx, req.NamespacedName, &release); err != nil {
		log.Error(err, "unable to fetch release")
		return ctrl.Result{}, ignoreNotFound(err)
	}

	return ctrl.Result{}, reconcileWithDirector(ctx, r.Client, r.Director, &release)
}

func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&boshv1.Release{}).
		Complete(r)
}
