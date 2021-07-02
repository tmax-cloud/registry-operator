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
	"github.com/Nerzal/gocloak/v7"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	exv1beta1 "k8s.io/api/extensions/v1beta1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/regctl"
)

// RegistryReconciler reconciles a Registry object
type RegistryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var keycloak gocloak.GoCloak

func init() {
	address := config.Config.GetString(config.ConfigKeycloakService)
	keycloak = gocloak.NewClient(address)
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

	reg := &regv1.Registry{}
	err := r.Get(ctx, req.NamespacedName, reg)
	if err != nil {
		if k8serr.IsNotFound(err) {
			username := config.Config.GetString("keycloak.username")
			password := config.Config.GetString("keycloak.password")

			token, err := keycloak.LoginAdmin(ctx, username, password, "master")
			if err != nil {
				logger.Error(err, "failed to login keycloak")
				return reconcile.Result{}, err
			}
			realmName := fmt.Sprintf("%s-%s", reg.Namespace, reg.Name)
			if err = keycloak.DeleteRealm(ctx, token.AccessToken, realmName); err != nil {
				logger.Error(err, "failed to delete realm")
				return reconcile.Result{}, err
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
		username := config.Config.GetString("keycloak.username")
		password := config.Config.GetString("keycloak.password")

		token, err := keycloak.LoginAdmin(ctx, username, password, "master")
		if err != nil {
			logger.Error(err, "failed to login keycloak")
			return reconcile.Result{}, err
		}

		realmName := fmt.Sprintf("%s-%s", reg.Namespace, reg.Name)
		enabled := true
		if _, err = keycloak.CreateRealm(ctx, token.AccessToken, gocloak.RealmRepresentation{
			ID:      &realmName,
			Realm:   &realmName,
			Enabled: &enabled,
		}); err != nil {
			logger.Error(err, "failed to create realm")
			return reconcile.Result{}, err
		}

		clientName := realmName + "-docker-client"
		protocol := "docker-v2"
		if _, err = keycloak.CreateClient(ctx, token.AccessToken, realmName, gocloak.Client{
			ClientID: &clientName,
			Protocol: &protocol,
		}); err != nil {
			logger.Error(err, "Couldn't create docker client in realm "+realmName)
			return reconcile.Result{}, err
		}

		//if err := c.AddCertificate(); err != nil {
		//	logger.Error(err, "failed to add a certificate")
		//	return reconcile.Result{}, err
		//}

		user, err := keycloak.CreateUser(ctx, token.AccessToken, realmName, gocloak.User{
			Username: &reg.Spec.LoginID,
			Enabled:  &enabled,
		})
		if err != nil {
			logger.Error(err, "failed to create user")
			return reconcile.Result{}, err
		}

		if err = keycloak.SetPassword(ctx, token.AccessToken, user, realmName, reg.Spec.LoginPassword, false); err != nil {
			logger.Error(err, "failed to set password")
			return reconcile.Result{}, err
		}

		// -------------------
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

	if requeue, err := r.handleAllSubresources(reg); err != nil {
		return reconcile.Result{}, err
	} else if requeue {
		return reconcile.Result{Requeue: true}, nil
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

func (r *RegistryReconciler) handleAllSubresources(reg *regv1.Registry) (bool, error) { // if want to requeue, return true
	logger := r.Log.WithValues("namespace", reg.Namespace, "name", reg.Name)
	gErrors := []error{}
	gRequeue := false

	defer func() {
		if err := r.Status().Update(context.TODO(), reg); err != nil {
			logger.Error(err, "failed to update condition")
		}
	}()

	components := r.getComponentControllerList(reg)
	for _, component := range components {
		requeue, err := component.ReconcileByConditionStatus(reg)
		if requeue {
			gRequeue = true
		}
		if err != nil {
			gErrors = append(gErrors, err)
			continue
		}
	}

	if len(gErrors) > 0 {
		eMsgs := []string{}
		for _, e := range gErrors {
			eMsgs = append(eMsgs, e.Error())
		}
		return gRequeue, fmt.Errorf(strings.Join(eMsgs, ", "))
	}
	return gRequeue, nil
}

func (r *RegistryReconciler) getComponentControllerList(reg *regv1.Registry) []regctl.ResourceController {
	logger := r.Log.WithValues("namespace", reg.Namespace, "name", reg.Name)

	serverAddr := config.Config.GetString(config.ConfigKeycloakService)
	realmName := fmt.Sprintf("%s-%s", reg.Namespace, reg.Name)
	clientName := realmName + "-docker-client"
	authcfg := &regv1.AuthConfig{
		Realm:   path.Join(serverAddr, "auth", "realms", realmName, "protocol", "docker-v2", "auth"),
		Service: clientName,
		Issuer:  path.Join(serverAddr, "auth", "realms", realmName),
	}

	collection := []regctl.ResourceController{}
	for _, cond := range reg.Status.Conditions {
		switch cond.Type {
		case regv1.ConditionTypeDeployment:
			collection = append(collection, regctl.NewRegistryDeployment(r.Client, func() (interface{}, error) {
				manifest, err := schemes.Deployment(reg, authcfg)
				if err != nil {
					return nil, err
				}
				if err = controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger).Require(regv1.ConditionTypeService).Require(regv1.ConditionTypePvc).Require(regv1.ConditionTypeConfigMap))
		case regv1.ConditionTypePod:
			collection = append(collection, regctl.NewRegistryPod(r.Client, func() (interface{}, error) {
				return nil, nil
			}, cond.Type, logger))
		case regv1.ConditionTypeContainer:
			collection = append(collection, regctl.NewRegistryContainer(r.Client, func() (interface{}, error) {
				return nil, nil
			}, cond.Type, logger).Require(regv1.ConditionTypeDeployment).Require(regv1.ConditionTypeDeployment).Require(regv1.ConditionTypePod))
		case regv1.ConditionTypeService:
			collection = append(collection, regctl.NewRegistryService(r.Client, func() (interface{}, error) {
				manifest := schemes.Service(reg)
				if err := controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger))
		case regv1.ConditionTypeSecretTLS:
			collection = append(collection, regctl.NewRegistryTlsCertSecret(r.Client, func() (interface{}, error) {
				manifest, err := schemes.TlsSecret(reg, r.Client)
				if err != nil {
					return nil, err
				}
				if err = controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger).Require(regv1.ConditionTypeService))
		case regv1.ConditionTypeSecretOpaque:
			collection = append(collection, regctl.NewRegistryCrendentialSecret(r.Client, func() (interface{}, error) {
				manifest := schemes.CredentialSecret(reg)
				if err := controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger).Require(regv1.ConditionTypeService))
		case regv1.ConditionTypeSecretDockerConfigJSON:
			collection = append(collection, regctl.NewRegistryDCJSecret(r.Client, func() (interface{}, error) {
				manifest := schemes.DCJSecret(reg)
				if err := controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger).Require(regv1.ConditionTypeService))
		case regv1.ConditionTypePvc:
			collection = append(collection, regctl.NewRegistryPVC(r.Client, func() (interface{}, error) {
				manifest := schemes.PersistentVolumeClaim(reg)
				if err := controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger))
		case regv1.ConditionTypeConfigMap:
			collection = append(collection, regctl.NewRegistryConfigMap(r.Client, func() (interface{}, error) {
				ctx := context.TODO()
				base := &corev1.ConfigMap{}
				if err := r.Get(ctx, types.NamespacedName{Name: "registry-config", Namespace: regv1.OperatorNamespace}, base); err != nil {
					return nil, err
				}
				manifest := schemes.ConfigMap(reg, base.Data)
				if err := controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger))
		case regv1.ConditionTypeNotary:
			collection = append(collection, regctl.NewRegistryNotary(r.Client, func() (interface{}, error) {
				manifest, err := schemes.Notary(reg, authcfg)
				if err != nil {
					return nil, err
				}
				if err = controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger))
		case regv1.ConditionTypeIngress:
			collection = append(collection, regctl.NewRegistryIngress(r.Client, func() (interface{}, error) {
				manifest := schemes.Ingress(reg)
				if err := controllerutil.SetControllerReference(reg, manifest, r.Scheme); err != nil {
					return nil, err
				}
				return manifest, nil
			}, cond.Type, logger).Require(regv1.ConditionTypeSecretTLS))
		default:
			logger.Info("[WARN] Unknown condition: " + string(cond.Type))
		}
	}

	return collection
}
