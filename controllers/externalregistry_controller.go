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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/registry/ext"
	harborv2 "github.com/tmax-cloud/registry-operator/pkg/registry/ext/harbor/v2"
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

	// get image signer
	exreg := &tmaxiov1.ExternalRegistry{}
	if err := r.Get(context.TODO(), req.NamespacedName, exreg); err != nil {
		log.Error(err, "")
		return ctrl.Result{}, nil
	}

	var syncClient ext.Synchronizable

	if exreg.Spec.RegistryType == tmaxiov1.RegistryTypeHarborV2 {
		basic, err := utils.GetBasicAuth(exreg.Spec.ImagePullSecret, exreg.Namespace, exreg.Spec.RegistryURL)
		if err != nil {
			r.Log.Error(err, "failed to get basic auth")
			return ctrl.Result{}, nil
		}

		username, password := utils.DecodeBasicAuth(basic)
		ca, err := utils.GetCAData(exreg.Spec.CertificateSecret, exreg.Namespace)
		if err != nil {
			r.Log.Error(err, "failed to get ca data")
			return ctrl.Result{}, nil
		}
		syncClient = harborv2.NewClient(r.Client, req.NamespacedName, r.Scheme, exreg.Spec.RegistryURL, username, password, ca, exreg.Spec.Insecure)

		if err := syncClient.Synchronize(); err != nil {
			r.Log.Error(err, "failed to synchronize external registry")
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *ExternalRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ExternalRegistry{}).
		Complete(r)
}
