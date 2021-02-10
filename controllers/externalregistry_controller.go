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
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/exregctl"
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
	_ = context.Background()
	_ = r.Log.WithValues("externalregistry", req.NamespacedName)

	// Fetch the External Registry reg
	reg := &regv1.ExternalRegistry{}
	err := r.Get(context.TODO(), req.NamespacedName, reg)
	if err != nil {
		r.Log.Info("Error on get registry")

		return ctrl.Result{}, err
	}

	updated, err := exregctl.UpdateRegistryStatus(r.Client, reg)
	if err != nil {
		return ctrl.Result{}, err
	} else if updated {
		return ctrl.Result{}, nil
	}

	if err = r.handleAllSubresources(reg); err != nil {
		r.Log.Error(err, "Subresource creation failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ExternalRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&regv1.ExternalRegistry{}).
		Owns(&regv1.RegistryCronJob{}).
		Complete(r)
}

func (r *ExternalRegistryReconciler) handleAllSubresources(exreg *regv1.ExternalRegistry) error { // if want to requeue, return true
	subResourceLogger := r.Log.WithValues("SubResource.Namespace", exreg.Namespace, "SubResource.Name", exreg.Name)
	subResourceLogger.Info("Creating all Subresources")

	patchReg := exreg.DeepCopy() // Target to Patch object

	defer func() {
		if err := r.update(exreg, patchReg); err != nil {
			subResourceLogger.Error(err, "failed to update")
		}
	}()

	collectSubController := collectExRegSubController(exreg)

	// Check if subresources are created.
	for _, sctl := range collectSubController {
		subresourceType := reflect.TypeOf(sctl).String()
		subResourceLogger.Info("Check subresource", "subresourceType", subresourceType)

		// Check if subresource is handled.
		if err := sctl.Handle(r.Client, exreg, patchReg, r.Scheme); err != nil {
			subResourceLogger.Error(err, "Got an error in creating subresource ")
			return err
		}

		// Check if subresource is ready.
		if err := sctl.Ready(r.Client, exreg, patchReg, false); err != nil {
			subResourceLogger.Error(err, "Got an error in checking ready")
			return err
		}
	}
	return nil
}

func (r *ExternalRegistryReconciler) update(origin, target *regv1.ExternalRegistry) error {
	subResourceLogger := r.Log.WithValues("SubResource.Namespace", origin.Namespace, "SubResource.Name", origin.Name)

	// Update spec, if patch exists
	if !reflect.DeepEqual(origin.Spec, target.Spec) {
		subResourceLogger.Info("Update registry")
		if err := r.Update(context.TODO(), target); err != nil {
			subResourceLogger.Error(err, "Unknown error updating")
			return err
		}
	}

	// Update status, if patch exists
	if !reflect.DeepEqual(origin.Status, target.Status) {
		if err := r.Status().Update(context.TODO(), target); err != nil {
			subResourceLogger.Error(err, "Unknown error updating status")
			return err
		}
	}

	return nil
}

func collectExRegSubController(exreg *regv1.ExternalRegistry) []exregctl.ExternalRegistrySubresource {
	collection := []exregctl.ExternalRegistrySubresource{}

	collection = append(collection, &exregctl.RegistryCronJob{}, &exregctl.RegistryJob{})

	return collection
}
