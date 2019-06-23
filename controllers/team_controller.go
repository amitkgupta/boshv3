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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	boshv1 "github.com/amitkgupta/boshv3/api/v1"
	"github.com/amitkgupta/boshv3/remote-clients"
)

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Log                 logr.Logger
	BOSHSystemNamespace string
}

// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=teams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bosh.akgupta.ca,resources=teams/status,verbs=get;update;patch

func (r *TeamReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("team", req.NamespacedName)

	var team boshv1.Team
	if err := r.Get(ctx, req.NamespacedName, &team); err != nil {
		log.Error(err, "Unable to fetch team")
		return ctrl.Result{}, ignoreDoesNotExist(err)
	}

	if team.PrepareToSave(r.BOSHSystemNamespace) {
		if err := r.Status().Update(ctx, &team); err != nil {
			return ctrl.Result{}, err
		}
	}

	if uc, err := uaaAdminForDirector(
		ctx,
		r.Client,
		team.Status.OriginalDirector,
		r.BOSHSystemNamespace,
	); err != nil {
		return ctrl.Result{}, err
	} else {
		return ctrl.Result{}, reconcileWithUAA(
			ctx,
			r.Client,
			uc,
			&team,
		)
	}
}

func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&boshv1.Team{}).
		Complete(r)
}

func uaaAdminForDirector(
	ctx context.Context,
	c client.Client,
	directorName string,
	boshSystemNamespace string,
) (remoteclients.UAAClient, error) {
	var director boshv1.Director
	if err := c.Get(
		ctx,
		types.NamespacedName{Name: directorName},
		&director,
	); err != nil {
		return nil, err
	}

	var directorSecret v1.Secret
	if err := c.Get(
		ctx,
		types.NamespacedName{
			Namespace: boshSystemNamespace,
			Name:      director.Spec.UAAClientSecret,
		},
		&directorSecret,
	); err != nil {
		return nil, err
	}

	return remoteclients.NewUAAClient(
		director.Spec.UAAURL,
		director.Spec.UAAClient,
		string(directorSecret.Data["secret"]),
		director.Spec.UAACACert,
	)
}

func reconcileWithUAA(
	ctx context.Context,
	c client.Client,
	uc remoteclients.UAAClient,
	ue uaaEntity,
) error {
	if ue.BeingDeleted() {
		if err := ue.DeleteIfExists(uc); err != nil {
			return err
		}

		if err := c.Delete(ctx, &(v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ue.SecretName(),
				Namespace: ue.SecretNamespace(),
			},
		})); ignoreDoesNotExist(err) != nil {
			return err
		}

		if ue.EnsureNoFinalizer() {
			if err := c.Update(ctx, ue); err != nil {
				return err
			}
		}

		return nil
	}

	if ue.EnsureFinalizer() {
		if err := c.Update(ctx, ue); err != nil {
			return err
		}
	}

	secretData := "TODOMAKEBETTERSECRET"
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ue.SecretName(),
			Namespace: ue.SecretNamespace(),
		},
		Type:       v1.SecretTypeOpaque,
		StringData: map[string]string{"secret": secretData},
	}
	if err := c.Create(ctx, &secret); ignoreAlreadyExists(err) != nil {
		return err
	}

	if err := ue.CreateUnlessExists(uc, secretData); err != nil {
		return err
	} else {
		return c.Status().Update(ctx, ue)
	}
}
