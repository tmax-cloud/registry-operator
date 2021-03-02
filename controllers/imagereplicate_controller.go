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

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/replicatectl/handler"
	"github.com/tmax-cloud/registry-operator/pkg/scheduler"
)

// ImageReplicateReconciler reconciles a ImageReplicate object
type ImageReplicateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=imagereplicates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=imagereplicates/status,verbs=get;update;patch

func (r *ImageReplicateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("imagereplicate", req.NamespacedName)

	// get image replicate
	r.Log.Info("get image replicate")
	replImage := &tmaxiov1.ImageReplicate{}
	if err := r.Get(context.TODO(), req.NamespacedName, replImage); err != nil {
		r.Log.Error(err, "")
		return ctrl.Result{}, nil
	}

	// TODO: update status

	// TODO: handle registry job

	return ctrl.Result{}, nil
}

func (r *ImageReplicateReconciler) SetupWithManager(mgr ctrl.Manager, s *scheduler.Scheduler) error {
	h := handler.NewReplicateHandler(mgr.GetClient(), mgr.GetScheme())
	if err := s.RegisterHandler(regv1.JobTypeImageReplicate, h); err != nil {
		r.Log.Error(err, "unable to register handler", "type", regv1.JobTypeImageReplicate)
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ImageReplicate{}).
		Complete(r)
}
