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
	"crypto/tls"
	"fmt"
	"github.com/Nerzal/gocloak/v7"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/regctl"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"path"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// RegistryReconciler reconciles a Registry object
type RegistryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var keycloak gocloak.GoCloak

func init() {
	address := config.Config.GetString(config.ConfigTokenServiceAddr)
	insecure := config.Config.GetBool(config.ConfigTokenServiceInsecure)
	keycloak = gocloak.NewClient(address)

	// Configure gocloak to skip TLS verification
	// FIXME: load value from manager_config.
	restyKeycloak := keycloak.RestyClient()
	restyKeycloak.SetDebug(false)
	restyKeycloak.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: insecure})
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

	o := &regv1.Registry{}
	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		if k8serr.IsNotFound(err) {
			username := config.Config.GetString("keycloak.username")
			password := config.Config.GetString("keycloak.password")

			token, err := keycloak.LoginAdmin(ctx, username, password, "master")
			if err != nil {
				logger.Error(err, "failed to login keycloak")
				return reconcile.Result{}, err
			}

			realmName := req.Namespace
			clientId := req.Name
			logger.Info("delete client info:", "realm", req.Namespace, "client", clientId)

			clients, err := keycloak.GetClients(ctx, token.AccessToken, realmName, gocloak.GetClientsParams{})
			if err != nil {
				logger.Error(err, "failed to get clients")
				return ctrl.Result{}, err
			}

			for _, c := range clients {
				if *c.ClientID == clientId {
					if err = keycloak.DeleteClient(ctx, token.AccessToken, realmName, *c.ID); err != nil {
						logger.Error(err, "failed to delete client")
						return ctrl.Result{}, err
					}
				}
			}

			// TODO: 더 이상 사용되지 않는 realm(namespace) 정리: 일정 주기로 realm을 감시하는 백그라운드 잡?
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// FIXME: move to validating webhook
	if err = r.validate(o); err != nil {
		return reconcile.Result{}, err
	}

	switch o.Status.Phase {
	case "":
		username := config.Config.GetString("keycloak.username")
		password := config.Config.GetString("keycloak.password")

		token, err := keycloak.LoginAdmin(ctx, username, password, "master")
		if err != nil {
			logger.Error(err, "failed to login keycloak")
			return reconcile.Result{}, err
		}

		enabled := true
		realmName := o.Namespace
		realm, err := keycloak.GetRealm(ctx, token.AccessToken, realmName)
		if err != nil {
			if apiError := err.(*gocloak.APIError); apiError.Code == http.StatusNotFound {
				_, err := keycloak.CreateRealm(ctx, token.AccessToken, gocloak.RealmRepresentation{
					ID:      &realmName,
					Realm:   &realmName,
					Enabled: &enabled,
				})
				if err != nil {
					logger.Error(err, "failed to create realm")
					return reconcile.Result{}, err
				}
			} else {
				return ctrl.Result{}, err
			}
		}
		if realm == nil {
			return ctrl.Result{}, fmt.Errorf("nil realm")
		}
		logger.Info("found realm", "realmID", realm.ID, "realm", realm.Realm)

		clientName := o.Name
		protocol := "docker-v2"
		isClientExist, err := func() (bool, error) {
			clients, err := keycloak.GetClients(ctx, token.AccessToken, *realm.Realm, gocloak.GetClientsParams{})
			if err != nil {
				logger.Error(err, "failed to get clients")
				return false, err
			}

			for _, c := range clients {
				if *c.ClientID == clientName {
					return true, nil
				}
			}
			return false, nil
		}()
		if err != nil {
			return reconcile.Result{}, err
		}

		if !isClientExist {
			created, err := keycloak.CreateClient(ctx, token.AccessToken, realmName, gocloak.Client{
				ClientID: &clientName,
				Protocol: &protocol,
			})
			if err != nil {
				logger.Error(err, "failed to create docker client")
				return reconcile.Result{}, err
			}
			logger.Info("client created: " + created)
		}

		//if err := c.AddCertificate(); err != nil {
		//	logger.Error(err, "failed to add a certificate")
		//	return reconcile.Result{}, err
		//}

		// ***** DO NOT CREATE USER *****
		//_, err = keycloak.GetUserByID(ctx, token.AccessToken, realmName, o.Spec.LoginID)
		//if err != nil {
		//	if apiError := err.(*gocloak.APIError); apiError.Code == http.StatusNotFound {
		//		created, err := keycloak.CreateUser(ctx, token.AccessToken, realmName, gocloak.User{
		//			Username: &o.Spec.LoginID,
		//			Enabled:  &enabled,
		//		})
		//		if err != nil {
		//			logger.Error(err, "failed to create user")
		//			return reconcile.Result{}, err
		//		}
		//
		//		logger.Info("user created: " + created)
		//		if err = keycloak.SetPassword(ctx, token.AccessToken, o.Spec.LoginID, realmName, o.Spec.LoginPassword, false); err != nil {
		//			logger.Error(err, "failed to set password")
		//			return reconcile.Result{}, err
		//		}
		//	} else {
		//		return ctrl.Result{}, err
		//	}
		//}

		typesToManage := []status.ConditionType{
			regv1.ConditionTypeConfigMap,
			regv1.ConditionTypeDeployment,
			regv1.ConditionTypeService,
			regv1.ConditionTypeSecretDockerConfigJSON,
			regv1.ConditionTypeSecretTLS,
			regv1.ConditionTypeSecretOpaque,
			regv1.ConditionTypePod,
			regv1.ConditionTypePvc,
		}
		if o.Spec.Notary.Enabled {
			typesToManage = append(typesToManage, regv1.ConditionTypeNotary)
		}
		if o.Spec.RegistryService.ServiceType == "Ingress" {
			typesToManage = append(typesToManage, regv1.ConditionTypeIngress)
		}

		conds := status.Conditions{}
		for _, t := range typesToManage {
			conds = append(conds, status.Condition{Type: t, Status: corev1.ConditionFalse})
		}
		// -------------------
		o.Status.Conditions = status.NewConditions(conds...)
		o.Status.Message = "registry is creating. All resources in registry has not yet been created."
		o.Status.Reason = "AllConditionsNotTrue"
		o.Status.Phase = regv1.StatusCreating
		o.Status.PhaseChangedAt = metav1.Now()
		if err := r.Status().Update(ctx, o); err != nil {
			logger.Error(err, "failed to initialize conditions.")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	case regv1.StatusRunning:
		return reconcile.Result{}, nil
	}

	requeue := false
	components := r.getComponentControllerList(o)
	for _, component := range components {
		if requeue, err = component.ReconcileByConditionStatus(o); err != nil {
			return reconcile.Result{}, err
		}
	}

	r.setPhaseByCondition(o)
	if err = r.Status().Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: requeue}, nil
}

func (r *RegistryReconciler) setPhaseByCondition(reg *regv1.Registry) {
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
	case len(badConditions) == 1 && badConditions[0] == regv1.ConditionTypePod:
		reg.Status.Phase = regv1.StatusNotReady
		reg.Status.Message = "Container not ready."
		reg.Status.Reason = "NotReady"
	case len(badConditions) > 1:
		reg.Status.Phase = regv1.StatusCreating
		reg.Status.Message = "Registry is creating. All resources in registry has not yet been created."
		reg.Status.Reason = "AllConditionsNotTrue"
	}
	reg.Status.PhaseChangedAt = metav1.Now()
}

func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&regv1.Registry{}).
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

func (r *RegistryReconciler) getComponentControllerList(reg *regv1.Registry) []regctl.ResourceController {
	logger := r.Log.WithValues("namespace", reg.Namespace, "name", reg.Name)

	serverAddr := config.Config.GetString(config.ConfigTokenServiceAddr)
	realmName := reg.Namespace
	clientName := reg.Name
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
