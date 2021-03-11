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
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/replicatectl"
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

	if replImage.Status.State == regv1.ImageReplicateSuccess ||
		replImage.Status.State == regv1.ImageReplicateFail {
		r.Log.Info("Image Replicate is already finished", "result", replImage.Status.State)
		return ctrl.Result{}, nil
	}

	updated, err := replicatectl.UpdateImageReplicateStatus(r.Client, replImage)
	if err != nil {
		return ctrl.Result{}, err
	} else if updated {
		return ctrl.Result{}, nil
	}

	if err = r.handleAllSubresources(replImage); err != nil {
		r.Log.Error(err, "Subresource creation failed")
		return ctrl.Result{}, err
	}

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
		Owns(&tmaxiov1.RegistryJob{}).
		Owns(&tmaxiov1.ImageSignRequest{}).
		Complete(r)
}

func (r *ImageReplicateReconciler) handleAllSubresources(repl *regv1.ImageReplicate) error { // if want to requeue, return true
	subResourceLogger := r.Log.WithValues("SubResource.Namespace", repl.Namespace, "SubResource.Name", repl.Name)
	subResourceLogger.Info("Creating all Subresources")

	patchRepl := repl.DeepCopy() // Target to Patch object

	defer func() {
		if err := r.update(repl, patchRepl); err != nil {
			subResourceLogger.Error(err, "failed to update")
		}
	}()

	collectSubController := collectImageReplicateSubController(repl)

	// Check if subresources are created.
	for _, sctl := range collectSubController {
		subresourceType := reflect.TypeOf(sctl).String()
		subResourceLogger.Info("Check subresource", "subresourceType", subresourceType)

		// Check if subresource is handled.
		if err := sctl.Handle(r.Client, repl, patchRepl, r.Scheme); err != nil {
			subResourceLogger.Error(err, "Got an error in creating subresource ")
			return err
		}

		// Check if subresource is ready.
		if err := sctl.Ready(r.Client, repl, patchRepl, false); err != nil {
			subResourceLogger.Error(err, "Got an error in checking ready")
			return err
		}
	}
	return nil
}

func (r *ImageReplicateReconciler) update(origin, target *regv1.ImageReplicate) error {
	subResourceLogger := r.Log.WithValues("SubResource.Namespace", origin.Namespace, "SubResource.Name", origin.Name)

	// Update spec, if patch exists
	if !reflect.DeepEqual(origin.Spec, target.Spec) {
		subResourceLogger.Info("Update image replicate")
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

func collectImageReplicateSubController(repl *regv1.ImageReplicate) []replicatectl.ImageReplicateSubresource {
	collection := []replicatectl.ImageReplicateSubresource{}

	registryJob := replicatectl.RegistryJob{}
	collection = append(collection, &registryJob)

	if repl.Spec.Signer != "" {
		collection = append(collection, replicatectl.NewImageSignRequest(&registryJob))
	}

	return collection
}
