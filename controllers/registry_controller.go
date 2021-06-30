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
	"fmt"
	"github.com/operator-framework/operator-lib/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	exv1beta1 "k8s.io/api/extensions/v1beta1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/keycloakctl"
	"github.com/tmax-cloud/registry-operator/controllers/regctl"
)

// RegistryReconciler reconciles a Registry object
type RegistryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	kc     *keycloakctl.KeycloakController
}

// +kubebuilder:rbac:groups=tmax.io,resources=registries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=registries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/exec,verbs=get;list;watch;create;update;patch;delete

func (r *RegistryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("registry", req.NamespacedName)

	logger.Info(">>>> Start reconcile...")
	// Fetch the Registry reg
	reg := &regv1.Registry{}
	err := r.Get(ctx, req.NamespacedName, reg)
	if err != nil {
		if k8serr.IsNotFound(err) {
			// Delete 되었을 때 이 로직이 반드시 수행되는가?
			logger.Info("****** DELETED registry *****")
			r.kc = keycloakctl.NewKeycloakController(req.Namespace, req.Name, logger)
			if r.kc == nil {
				return reconcile.Result{}, err
			}
			if err := r.kc.DeleteRealm(req.Namespace, req.Name); err != nil {
				logger.Info("Couldn't delete keycloak realm")
			}
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// FIXME: move to validating webhook
	if err = r.validate(reg); err != nil {
		return reconcile.Result{}, err
	}

	if reg.Status.Phase == "" {
		reg.Status.Conditions = status.NewConditions(getResourceConditionList(reg)...)
		reg.Status.Message = "registry is creating. All resources in registry has not yet been created."
		reg.Status.Reason = "AllConditionsNotTrue"
		reg.Status.Phase = regv1.StatusCreating
		reg.Status.PhaseChangedAt = metav1.Now()
		if err := r.Status().Update(ctx, reg); err != nil {
			logger.Error(err, "failed to initialize conditions.")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Check if all subresources are true
	if err := r.updatePhaseByCondition(ctx, reg); err != nil {
		logger.Error(err, "failed to update")
		return reconcile.Result{}, err
	}

	logger.Info("Processing resources...")
	if err = r.handleAllSubresources(reg); err != nil {
		logger.Error(err, "Subresource creation failed")
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func getResourceConditionList(reg *regv1.Registry) status.Conditions {
	condTypes := []status.ConditionType{
		regv1.ConditionTypeDeployment,
		regv1.ConditionTypePod,
		regv1.ConditionTypeContainer,
		regv1.ConditionTypeService,
		regv1.ConditionTypeSecretTLS,
		regv1.ConditionTypeSecretOpaque,
		regv1.ConditionTypeSecretDockerConfigJSON,
		regv1.ConditionTypePvc,
		regv1.ConditionTypeConfigMap,
		regv1.ConditionTypeKeycloakResources,
	}
	if reg.Spec.Notary.Enabled {
		condTypes = append(condTypes, regv1.ConditionTypeNotary)
	}
	if reg.Spec.RegistryService.ServiceType == "Ingress" {
		condTypes = append(condTypes, regv1.ConditionTypeIngress)
	}

	conds := status.Conditions{}
	for _, t := range condTypes {
		conds = append(conds, status.Condition{Type: t, Status: corev1.ConditionFalse})
	}
	return conds
}

func (r *RegistryReconciler) updatePhaseByCondition(ctx context.Context, reg *regv1.Registry) error {
	badConditions := []status.ConditionType{}
	for _, cond := range reg.Status.Conditions {
		if reg.Status.Conditions.IsFalseFor(cond.Type) {
			badConditions = append(badConditions, cond.Type)
		}
	}

	switch {
	case len(badConditions) == 0:
		reg.Status.Phase = regv1.StatusRunning
		reg.Status.Message = "Registry is running. All registry resources are operating normally."
		reg.Status.Reason = "Running"
	case len(badConditions) == 1 && badConditions[0] == regv1.ConditionTypeContainer:
		reg.Status.Phase = regv1.StatusNotReady
		reg.Status.Message = "Registry is not ready."
		reg.Status.Reason = "NotReady"
	case len(badConditions) > 1:
		reg.Status.Phase = regv1.StatusCreating
		reg.Status.Message = "Registry is creating. All resources in registry has not yet been created."
		reg.Status.Reason = "AllConditionsNotTrue"
	}
	reg.Status.PhaseChangedAt = metav1.Now()
	if err := r.Status().Update(ctx, reg); err != nil {
		return err
	}
	return nil
}

func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&regv1.Registry{}).
		Owns(&regv1.Notary{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&exv1beta1.Ingress{}).
		Complete(r)
}

func (r *RegistryReconciler) validate(reg *regv1.Registry) error {
	// this is for checking if field is empty
	emptyPvc := regv1.NotaryPVC{}
	if reg.Spec.Notary.Enabled &&
		(len(reg.Spec.Notary.ServiceType) == 0 || reg.Spec.Notary.PersistentVolumeClaim == emptyPvc) {
		return fmt.Errorf("notary's service type or pvc field missing")
	}
	return nil
}

func (r *RegistryReconciler) handleAllSubresources(reg *regv1.Registry) error { // if want to requeue, return true
	logger := r.Log.WithValues("SubResource.Namespace", reg.Namespace, "SubResource.Name", reg.Name)
	logger.Info("Creating all Subresources")

	var requeueErr error
	patchReg := reg.DeepCopy() // Target to Patch object

	defer func() {
		if err := r.update(reg, patchReg); err != nil {
			logger.Error(err, "failed to patch")
		}
	}()

	r.kc = keycloakctl.NewKeycloakController(reg.Namespace, reg.Name, logger)
	if r.kc == nil {
		return fmt.Errorf("unable to get keycloak controller")
	}
	if err := r.kc.CreateResources(reg, patchReg); err != nil {
		return err
	}
	collectSubController := r.collectSubController(reg)
	printSubresources(log, collectSubController)

	// Check if subresources are created.
	for _, sctl := range collectSubController {
		subresourceType := reflect.TypeOf(sctl).String()
		logger.Info("Check subresource", "subresourceType", subresourceType)

		// Check if subresource is handled.
		if err := sctl.CreateIfNotExist(reg, patchReg); err != nil {
			errMsg := "Got an error in handling subresource"
			logger.Error(err, errMsg)
			requeueErr = regv1.AppendError(requeueErr, errMsg)
			continue
		}

		// Check if subresource is ready.
		if err := sctl.IsReady(reg, patchReg, true); err != nil {
			errMsg := "Got an error in checking ready"
			logger.Error(err, errMsg)
			requeueErr = regv1.AppendError(requeueErr, errMsg)
		}
	}

	if requeueErr != nil {
		return requeueErr
	}

	return nil
}

func (r *RegistryReconciler) update(origin, target *regv1.Registry) error {
	logger := r.Log.WithValues("SubResource.Namespace", origin.Namespace, "SubResource.Name", origin.Name)

	// Check whether update is necessary or not
	if !reflect.DeepEqual(origin.Spec, target.Spec) {
		logger.Info("Update registry")
		if err := r.Update(context.TODO(), target); err != nil {
			logger.Error(err, "Unknown error updating")
			return err
		}
	}

	// Check whether update is necessary or not about status
	// r.exceptStatus(origin, target)
	if !reflect.DeepEqual(origin.Status, target.Status) {
		if err := r.Status().Update(context.TODO(), target); err != nil {
			logger.Error(err, "Unknown error updating status")
			return err
		}
	}

	return nil
}

func (r *RegistryReconciler) collectSubController(reg *regv1.Registry) []regctl.RegistrySubresource {
	logger := r.Log.WithValues("Namespace", reg.Namespace, "Name", reg.Name)

	notary := regctl.NewRegistryNotary(r.Client, r.Scheme, reg, logger, r.kc)
	pvc := regctl.NewRegistryPVC(r.Client, r.Scheme, reg, logger)
	svc := regctl.NewRegistryService(r.Client, r.Scheme, reg, logger)
	certSecret := regctl.NewRegistryCertSecret(r.Client, r.Scheme, reg, logger, svc)
	dcjSecret := regctl.NewRegistryDCJSecret(r.Client, r.Scheme, reg, logger, svc)
	cm := regctl.NewRegistryConfigMap(r.Client, r.Scheme, reg, logger)
	deploy := regctl.NewRegistryDeployment(r.Client, r.Scheme, reg, logger, r.kc, pvc, svc, cm)
	pod := regctl.NewRegistryPod(r.Client, r.Scheme, reg, logger, deploy)
	ing := regctl.NewRegistryIngress(r.Client, r.Scheme, reg, logger, certSecret)

	collection := []regctl.RegistrySubresource{}
	collection = append(collection, notary, pvc, svc, certSecret, dcjSecret, cm, deploy, pod, ing)
	return collection
}

func printSubresources(log logr.Logger, subresources []regctl.RegistrySubresource) {
	var printStr []string
	for _, res := range subresources {
		switch res.(type) {
		case *regctl.RegistryNotary:
			printStr = append(printStr, "RegistryNotary")
		case *regctl.RegistryPVC:
			printStr = append(printStr, "RegistryPVC")
		case *regctl.RegistryService:
			printStr = append(printStr, "RegistryService")
		case *regctl.RegistryCertSecret:
			printStr = append(printStr, "RegistryCertSecret")
		case *regctl.RegistryDCJSecret:
			printStr = append(printStr, "RegistryDCJSecret")
		case *regctl.RegistryConfigMap:
			printStr = append(printStr, "RegistryConfigMap")
		case *regctl.RegistryDeployment:
			printStr = append(printStr, "RegistryDeployment")
		case *regctl.RegistryPod:
			printStr = append(printStr, "RegistryPod")
		case *regctl.RegistryIngress:
			printStr = append(printStr, "RegistryIngress")
		}
	}

	log.Info("debug", "subresources", printStr)
}
