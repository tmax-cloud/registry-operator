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
	"bytes"
	"context"
	"encoding/json"
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	exv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/regctl"
)

// RegistryReconciler reconciles a Registry object
type RegistryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	kc     *regctl.KeycloakController
}

// +kubebuilder:rbac:groups=tmax.io,resources=registries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=registries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

func (r *RegistryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("registry", req.NamespacedName)

	// Fetch the Registry reg
	reg := &regv1.Registry{}
	err := r.Get(context.TODO(), req.NamespacedName, reg)
	if err != nil {
		r.Log.Info("Error on get registry")
		if errors.IsNotFound(err) {
			r.kc = regctl.NewKeycloakController(req.Namespace, req.Name)
			if err := r.kc.DeleteRealm(req.Namespace, req.Name); err != nil {
				r.Log.Info("Couldn't delete keycloak realm")
			}

			r.Log.Info("Not Found Error")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if regctl.UpdateRegistryStatus(r.Client, reg) {
		return reconcile.Result{}, nil
	}

	if err = r.handleAllSubresources(reg); err != nil {
		r.Log.Error(err, "Subresource creation failed")
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&regv1.Registry{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&exv1beta1.Ingress{}).
		Complete(r)
}

func (r *RegistryReconciler) handleAllSubresources(reg *regv1.Registry) error { // if want to requeue, return true
	subResourceLogger := r.Log.WithValues("SubResource.Namespace", reg.Namespace, "SubResource.Name", reg.Name)
	subResourceLogger.Info("Creating all Subresources")

	var requeueErr error = nil
	patchReg := reg.DeepCopy() // Target to Patch object

	defer r.patch(reg, patchReg)

	r.kc = regctl.NewKeycloakController(reg.Namespace, reg.Name)
	if reg.Status.Conditions.IsFalseFor(regv1.ConditionTypeKeycloakRealm) {
		if err := r.kc.CreateRealm(reg.Namespace, reg.Name, patchReg); err != nil {
			return err
		}
	}

	collectSubController := collectSubController(reg, r.kc)

	// Check if subresources are created.
	for _, sctl := range collectSubController {
		subresourceType := reflect.TypeOf(sctl).String()
		subResourceLogger.Info("Check subresource", "subresourceType", subresourceType)

		// Check if subresource is handled.
		if err := sctl.Handle(r.Client, reg, patchReg, r.Scheme); err != nil {
			subResourceLogger.Error(err, "Got an error in creating subresource ")
			return err
		}

		// Check if subresource is ready.
		if err := sctl.Ready(r.Client, reg, patchReg, false); err != nil {
			subResourceLogger.Error(err, "Got an error in checking ready")
			if regv1.IsPodError(err) {
				requeueErr = err
			} else {
				return err
			}
		}
	}

	if requeueErr != nil {
		return requeueErr
	}

	return nil
}

func (r *RegistryReconciler) patch(origin, target *regv1.Registry) error {
	subResourceLogger := r.Log.WithValues("SubResource.Namespace", origin.Namespace, "SubResource.Name", origin.Name)

	originObject := client.MergeFrom(origin) // Set original obeject
	statusPatchTarget := target.DeepCopy()

	// Get origin data except status for compare
	originWithoutStatus := origin.DeepCopy()
	originWithoutStatus.Status = regv1.RegistryStatus{}
	originWithoutStatusByte, err := json.Marshal(*originWithoutStatus)
	if err != nil {
		subResourceLogger.Error(err, "json marshal error")
		return err
	}

	// Get target data except status for compare
	targetWithoutStatus := target.DeepCopy()
	targetWithoutStatus.Status = regv1.RegistryStatus{}
	targetWithoutStatusByte, err := json.Marshal(*targetWithoutStatus)
	if err != nil {
		subResourceLogger.Error(err, "json marshal error")
		return err
	}

	// Check whether patch is necessary or not
	if res := bytes.Compare(originWithoutStatusByte, targetWithoutStatusByte); res != 0 {
		subResourceLogger.Info("Patch registry")
		if err := r.Patch(context.TODO(), target, originObject); err != nil {
			subResourceLogger.Error(err, "Unknown error patching status")
			return err
		}
	}

	// Get origin status data for compare
	originStatus := origin.Status.DeepCopy()
	originStatusByte, err := json.Marshal(*originStatus)
	if err != nil {
		subResourceLogger.Error(err, "json marshal error")
		return err
	}

	// Get target status data for compare
	targetStatusByte, err := json.Marshal(*statusPatchTarget)
	if err != nil {
		subResourceLogger.Error(err, "json marshal error")
		return err
	}

	// Check whether patch is necessary or not about status
	if res := bytes.Compare(originStatusByte, targetStatusByte); res != 0 {
		subResourceLogger.Info("Patch registry status")
		if err := r.Status().Patch(context.TODO(), statusPatchTarget, originObject); err != nil {
			subResourceLogger.Error(err, "Unknown error patching status")
			return err
		}
	}

	return nil
}

func collectSubController(reg *regv1.Registry, kc *regctl.KeycloakController) []regctl.RegistrySubresource {
	collection := []regctl.RegistrySubresource{}

	if reg.Spec.Notary.Enabled {
		collection = append(collection, &regctl.RegistryNotary{KcCtl: kc})
	}

	collection = append(collection, &regctl.RegistryPVC{}, &regctl.RegistryService{}, &regctl.RegistryCertSecret{},
		&regctl.RegistryDCJSecret{}, &regctl.RegistryConfigMap{}, &regctl.RegistryDeployment{KcCtl: kc}, &regctl.RegistryPod{})

	if reg.Spec.RegistryService.ServiceType == "Ingress" {
		collection = append(collection, &regctl.RegistryIngress{})
	}

	return collection
}
