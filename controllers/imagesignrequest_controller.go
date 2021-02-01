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
	"strings"

	exv1beta1 "k8s.io/api/extensions/v1beta1"

	"github.com/tmax-cloud/registry-operator/controllers/repoctl"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/schemes"

	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	controller "github.com/tmax-cloud/registry-operator/pkg/controllers"
	"github.com/tmax-cloud/registry-operator/pkg/trust"
)

const (
	DefaultHarborCoreIngress   = "tmax-harbor-ingress"
	DefaultHarborNotaryIngress = "tmax-harbor-ingress-notary"
	DefaultHarborNamespace     = "harbor"
)

// ImageSignRequestReconciler reconciles a ImageSignRequest object
type ImageSignRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=imagesignrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=imagesignrequests/status,verbs=get;update;patch

// +kubebuilder:rbac:groups=tmax.io,resources=signerkeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=signerkeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apiregistration.k8s.io,resourceNames=v1.registry.tmax.io,resources=apiservices,verbs=get;update;patch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resourceNames=registry-operator-webhook-cfg,resources=mutatingwebhookconfigurations,verbs=get;update;patch
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resourceNames=extension-apiserver-authentication,resources=configmaps,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func (r *ImageSignRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("imagesignrequest", req.NamespacedName)

	// get image sign request
	log.Info("get image sign request")
	signReq := &tmaxiov1.ImageSignRequest{}
	if err := r.Get(context.TODO(), req.NamespacedName, signReq); err != nil {
		log.Error(err, "")
		return ctrl.Result{}, nil
	}

	defer func() {
		if err := response(r.Client, signReq); err != nil {
			log.Error(err, "")
		}
	}()

	if signReq.Status.ImageSignResponse == nil {
		makeInitResponse(signReq)
		return ctrl.Result{}, nil
	}

	if signReq.Status.ImageSignResponse != nil && signReq.Status.ImageSignResponse.Result != regv1.ResponseResultSigning {
		return ctrl.Result{}, nil
	}

	// get image signer
	log.Info("get image signer")
	signer := &tmaxiov1.ImageSigner{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: signReq.Spec.Signer}, signer); err != nil {
		log.Error(err, "")
		makeResponse(signReq, false, err.Error(), "")
		return ctrl.Result{}, nil
	}

	// get sign key
	log.Info("get sign key")
	signerKey := &tmaxiov1.SignerKey{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: signReq.Spec.Signer}, signerKey); err != nil {
		log.Error(err, "")
		makeResponse(signReq, false, err.Error(), "")
		return ctrl.Result{}, nil
	}

	// Get secret
	regSecret := &corev1.Secret{}
	if signReq.Spec.DcjSecretName != "" {
		if err := r.Get(context.TODO(), types.NamespacedName{Name: signReq.Spec.DcjSecretName, Namespace: signReq.Namespace}, regSecret); err != nil {
			log.Error(err, "")
			makeResponse(signReq, false, err.Error(), "")
			return ctrl.Result{}, nil
		}
	}

	regCert := &corev1.Secret{}
	var ca []byte
	if signReq.Spec.CertSecretName != "" {
		if err := r.Get(context.TODO(), types.NamespacedName{Name: signReq.Spec.CertSecretName, Namespace: signReq.Namespace}, regCert); err != nil {
			log.Error(err, "")
			makeResponse(signReq, false, err.Error(), "")
			return ctrl.Result{}, nil
		}
		ca = regCert.Data[schemes.TLSCert]
	}

	// Start signing procedure
	img, err := trust.NewImage(signReq.Spec.Image, "", "", "", ca)
	if err != nil {
		log.Error(err, "")
		makeResponse(signReq, false, err.Error(), "")
		return ctrl.Result{}, nil
	}

	// Check if it's Harbor registry
	isHarbor := false
	regIng := &exv1beta1.Ingress{}
	harborNamespace := config.Config.GetString(config.ConfigHarborNamespace)
	if harborNamespace == "" {
		harborNamespace = DefaultHarborNamespace
	}

	harborCoreIngress := config.Config.GetString(config.ConfigHarborCoreIngress)
	if harborCoreIngress == "" {
		harborCoreIngress = DefaultHarborCoreIngress
	}

	harborNotaryIngress := config.Config.GetString(config.ConfigHarborNotaryIngress)
	if harborNotaryIngress == "" {
		harborNotaryIngress = DefaultHarborNotaryIngress
	}

	if err := r.Client.Get(context.Background(), types.NamespacedName{Name: harborCoreIngress, Namespace: harborNamespace}, regIng); err != nil {
		log.Error(err, "")
	}
	if regIng.ResourceVersion != "" && len(regIng.Spec.Rules) == 1 && img.Host == regIng.Spec.Rules[0].Host {
		isHarbor = true

		notIng := &exv1beta1.Ingress{}
		if err := r.Client.Get(context.Background(), types.NamespacedName{Name: harborNotaryIngress, Namespace: harborNamespace}, notIng); err != nil {
			log.Error(err, "")
			makeResponse(signReq, false, err.Error(), "")
			return ctrl.Result{}, nil
		}
		if len(notIng.Spec.Rules) == 0 {
			err := fmt.Errorf("harbor notary ingress is misconfigured")
			log.Error(err, "")
			makeResponse(signReq, false, err.Error(), "")
			return ctrl.Result{}, nil
		}

		coreScheme := "https"
		if len(regIng.Spec.TLS) == 0 {
			coreScheme = "http"
		}
		img.ServerUrl = fmt.Sprintf("%s://%s", coreScheme, regIng.Spec.Rules[0].Host)

		notScheme := "https"
		if len(notIng.Spec.TLS) == 0 {
			notScheme = "http"
		}
		img.NotaryServerUrl = fmt.Sprintf("%s://%s", notScheme, notIng.Spec.Rules[0].Host)
	}

	var targetReg *regv1.Registry
	// List registries and filter target registry - if it's not harbor registry
	if !isHarbor {
		log.Info("list registries")
		targetReg, err = r.findRegistryByHost(img.Host)
		if err != nil {
			log.Error(err, "")
			makeResponse(signReq, false, err.Error(), "")
			return ctrl.Result{}, nil
		}

		// Initialize Sign controller
		signCtl := controller.NewSigningController(r.Client, r.Scheme, signer, targetReg.Name, targetReg.Namespace)
		img.ServerUrl = signCtl.Regctl.GetEndpoint()
		img.NotaryServerUrl = signCtl.Regctl.GetNotaryEndpoint()

		// Verify if registry is valid now
		if len(img.ServerUrl) == 0 {
			makeResponse(signReq, false, "RegistryMisconfigured", "serverUrl is not set for the registry")
			return ctrl.Result{}, nil
		}
		if len(img.NotaryServerUrl) == 0 {
			makeResponse(signReq, false, "RegistryMisconfigured", "notaryUrl is not set for the registry, maybe notary is disabled for the registry")
			return ctrl.Result{}, nil
		}
	}

	if regSecret.ResourceVersion != "" {
		basicAuth, err := utils.ParseBasicAuth(regSecret, img.Host)
		if err != nil {
			log.Error(err, "")
			makeResponse(signReq, false, err.Error(), "")
			return ctrl.Result{}, nil
		}
		img.BasicAuth = basicAuth
	}

	// Sign image
	log.Info("sign image")
	signCtl := controller.NewSigningController(r.Client, r.Scheme, signer, "", "")
	if err := signCtl.SignImage(signerKey, img, ca); err != nil {
		log.Error(err, "sign image")
		makeResponse(signReq, false, err.Error(), "")
		return ctrl.Result{}, nil
	}

	if !isHarbor {
		// Update repository with signer
		log.Info(fmt.Sprintf("update repository with signer %s", signer.Name))
		repoCtl := repoctl.New()
		repo, err := repoCtl.Get(r.Client, targetReg, img.Name)
		if err != nil {
			log.Error(err, fmt.Sprintf("failed to update repository with signer %s", signer.Name))
			makeResponse(signReq, false, err.Error(), "")
			return ctrl.Result{}, nil
		}

		for i, v := range repo.Spec.Versions {
			if v.Version == img.Tag {
				repo.Spec.Versions[i].Signer = signer.Name
				break
			}
		}

		if err := repoCtl.Update(r.Client, repo); err != nil {
			log.Error(err, fmt.Sprintf("failed to update repository with signer %s", signer.Name))
			makeResponse(signReq, false, err.Error(), "")
		}
	}

	makeResponse(signReq, true, "", "")
	return ctrl.Result{}, nil
}

func (r *ImageSignRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ImageSignRequest{}).
		Complete(r)
}

func (r *ImageSignRequestReconciler) findRegistryByHost(hostname string) (*tmaxiov1.Registry, error) {
	regList := &tmaxiov1.RegistryList{}
	if err := r.List(context.TODO(), regList); err != nil {
		return nil, err
	}

	var targetReg tmaxiov1.Registry
	targetFound := false
	for _, r := range regList.Items {
		log.Info(r.Name)
		serverUrl := strings.TrimPrefix(r.Status.ServerURL, "https://")
		serverUrl = strings.TrimPrefix(serverUrl, "http://")
		serverUrl = strings.TrimSuffix(serverUrl, "/")

		if serverUrl == hostname {
			targetReg = r
			targetFound = true
		}
	}

	if !targetFound {
		return nil, fmt.Errorf("target registry is not an internal registry")
	}

	return &targetReg, nil
}
