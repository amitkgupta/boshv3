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
	// "errors"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
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

	originalSpec := release.Status.OriginalSpec

	if release.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(release.ObjectMeta.Finalizers, releaseFinalizer) {
			release.ObjectMeta.Finalizers = append(release.ObjectMeta.Finalizers, releaseFinalizer)
			if err := r.Update(ctx, &release); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(release.ObjectMeta.Finalizers, releaseFinalizer) {
			if err := r.deleteReleaseFromDirector(originalSpec); err != nil {
				return ctrl.Result{}, err
			}

			release.ObjectMeta.Finalizers = removeString(release.ObjectMeta.Finalizers, releaseFinalizer)
			if err := r.Update(ctx, &release); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	if originalSpec.Empty() {
		release.Status.OriginalSpec = release.Spec
		if err := r.Status().Update(ctx, &release); err != nil {
			log.Error(err, "unable to preserve original spec")
			return ctrl.Result{}, err
		}

		originalSpec = release.Spec
	}

	if originalSpec.Matches(release.Spec) && release.Status.Warning != "" {
		release.Status.Warning = ""
		if err := r.Status().Update(ctx, &release); err != nil {
			log.Error(err, "unable to unset mutation warning")
			return ctrl.Result{}, err
		}
	} else if !originalSpec.Matches(release.Spec) && release.Status.Warning != releaseMutationWarning {
		release.Status.Warning = releaseMutationWarning
		if err := r.Status().Update(ctx, &release); err != nil {
			log.Error(err, "unable to set mutation warning")
			return ctrl.Result{}, err
		}
	}

	if has, err := r.directorHasRelease(originalSpec); err != nil {
		log.Error(err, "failed to check if Director has release")
		return ctrl.Result{}, err
	} else if !has {
		if release.Status.PresentOnDirector {
			release.Status.PresentOnDirector = false
			if err := r.Status().Update(ctx, &release); err != nil {
				log.Error(err, "failed to indicate absence of release on Director")
				return ctrl.Result{}, err
			}
		}

		if err := r.directorFetchRelease(originalSpec); err != nil {
			log.Error(err, "failed to have Director fetch release")
			return ctrl.Result{}, err
		}
	}

	release.Status.PresentOnDirector = true
	if err := r.Status().Update(ctx, &release); err != nil {
		log.Error(err, "failed to indicate presence of release on Director")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&boshv1.Release{}).
		Complete(r)
}

const releaseFinalizer = "release.finalizers.bosh.akgupta.ca"

const releaseMutationWarning = "resource has been mutated; all changes ignored"

func (r *ReleaseReconciler) deleteReleaseFromDirector(rs boshv1.ReleaseSpec) error {
	if hasRelease, err := r.directorHasRelease(rs); err != nil {
		return err
	} else if !hasRelease {
		return nil
	}

	if release, err := r.Director.FindRelease(boshdir.NewReleaseSlug(rs.ReleaseName, rs.Version)); err != nil {
		return err
	} else {
		return release.Delete(false)
	}
}

func (r *ReleaseReconciler) directorHasRelease(rs boshv1.ReleaseSpec) (bool, error) {
	return r.Director.HasRelease(rs.ReleaseName, rs.Version, boshdir.OSVersionSlug{})
}

func (r *ReleaseReconciler) directorFetchRelease(rs boshv1.ReleaseSpec) error {
	return r.Director.UploadReleaseURL(rs.URL, rs.SHA1, false, false)
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
