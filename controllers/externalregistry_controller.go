/*


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
	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/exregctl"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ExternalRegistryReconciler reconciles a ExternalRegistry object
type ExternalRegistryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=externalregistries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=externalregistries/status,verbs=get;update;patch

func (r *ExternalRegistryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("externalregistry", req.NamespacedName)

	o := &regv1.ExternalRegistry{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		logger.Info("Error on get registry")
		return ctrl.Result{}, err
	}

	switch o.Status.State {
	case "":
		typesToManage := []status.ConditionType{
			regv1.ConditionTypeExRegistryCronJobExist,
			regv1.ConditionTypeExRegistryInitialized,
			regv1.ConditionTypeExRegistryLoginSecretExist,
		}
		conds := status.Conditions{}
		for _, t := range typesToManage {
			conds = append(conds, status.Condition{Type: t, Status: corev1.ConditionFalse})
		}

		o.Status.Conditions = conds
		o.Status.State = regv1.ExternalRegistryNotReady
		o.Status.StateChangedAt = metav1.Now()
		if err = r.Status().Update(ctx, o); err != nil {
			logger.Error(err, "failed to update status")
			return ctrl.Result{}, err
		}

	case regv1.ExternalRegistryNotReady:
		requeue := false
		for _, c := range r.getComponentControllerList(o) {
			if requeue, err = c.ReconcileByConditionStatus(o); err != nil {
				return reconcile.Result{}, err
			}
		}
		if err = r.Status().Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: requeue}, nil

	case regv1.ExternalRegistryReady:
		return ctrl.Result{}, nil

	default:
		logger.Info("unknown condition type.")
	}
	return ctrl.Result{}, nil
}

func (r *ExternalRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&regv1.ExternalRegistry{}).
		Owns(&corev1.Secret{}).
		Owns(&regv1.RegistryCronJob{}).
		Complete(r)
}

func (r *ExternalRegistryReconciler) getComponentControllerList(exreg *regv1.ExternalRegistry) []exregctl.ResourceController {
	logger := r.Log.WithValues("namespace", exreg.Namespace, "name", exreg.Name)

	collection := []exregctl.ResourceController{}
	for _, cond := range exreg.Status.Conditions {
		switch cond.Type {
		case regv1.ConditionTypeExRegistryCronJobExist:
			collection = append(collection, exregctl.NewRegistryCronJob(r.Client, func() (interface{}, error) {
				manifest := schemes.ExternalRegistryCronJob(exreg)
				if err := controllerutil.SetControllerReference(exreg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger))
		case regv1.ConditionTypeExRegistryInitialized:
			collection = append(collection, exregctl.NewRegistryJob(r.Client, func() (interface{}, error) {
				manifest := schemes.ExternalRegistryJob(exreg)
				if err := controllerutil.SetControllerReference(exreg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger).Require(regv1.ConditionTypeExRegistryLoginSecretExist))
		case regv1.ConditionTypeExRegistryLoginSecretExist:
			collection = append(collection, exregctl.NewLoginSecret(r.Client, func() (interface{}, error) {
				manifest, err := schemes.ExternalRegistryLoginSecret(exreg)
				if err != nil {
					return nil, err
				}
				if err = controllerutil.SetControllerReference(exreg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger))
		default:
			logger.Info("[WARN] Unknown condition: " + string(cond.Type))
		}
	}

	return collection
}
