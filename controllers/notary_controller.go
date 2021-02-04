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
	corev1 "k8s.io/api/core/v1"
	exv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/notaryctl"
)

// NotaryReconciler reconciles a Notary object
type NotaryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=notaries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=notaries/status,verbs=get;update;patch

func (r *NotaryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("notary", req.NamespacedName)

	notary := &regv1.Notary{}
	err := r.Get(context.TODO(), req.NamespacedName, notary)
	if err != nil {
		r.Log.Info("Error on get notary")
		if errors.IsNotFound(err) {
			r.Log.Info("Not Found Error")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	updated, err := notaryctl.UpdateNotaryStatus(r.Client, notary)
	if err != nil {
		return reconcile.Result{}, err
	} else if updated {
		return reconcile.Result{}, nil
	}

	if err = r.handleAllSubresources(notary); err != nil {
		r.Log.Error(err, "Subresource creation failed")
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NotaryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.Notary{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&exv1beta1.Ingress{}).
		Complete(r)
}

func (r *NotaryReconciler) handleAllSubresources(notary *regv1.Notary) error { // if want to requeue, return true
	subResourceLogger := r.Log.WithValues("SubResource.Namespace", notary.Namespace, "SubResource.Name", notary.Name)
	subResourceLogger.Info("Creating all Subresources")

	var requeueErr error = nil
	collectSubController := collectNotarySubController(notary.Spec.ServiceType)
	patchNotary := notary.DeepCopy() // Target to Patch object

	defer func() {
		if err := r.patch(notary, patchNotary); err != nil {
			subResourceLogger.Error(err, "failed to patch")
		}
	}()

	// Check if subresources are created.
	for _, sctl := range collectSubController {
		subresourceType := reflect.TypeOf(sctl).String()
		subResourceLogger.Info("Check subresource", "subresourceType", subresourceType)

		// Check if subresource is handled.
		if err := sctl.Handle(r.Client, notary, patchNotary, r.Scheme); err != nil {
			subResourceLogger.Error(err, "Got an error in creating subresource ")
			return err
		}

		// Check if subresource is ready.
		if err := sctl.Ready(r.Client, notary, patchNotary, false); err != nil {
			subResourceLogger.Error(err, "Got an error in checking ready")
			return err
		}
	}
	if requeueErr != nil {
		return requeueErr
	}

	return nil
}

func (r *NotaryReconciler) patch(origin, target *regv1.Notary) error {
	subResourceLogger := r.Log.WithValues("SubResource.Namespace", origin.Namespace, "SubResource.Name", origin.Name)
	originObject := client.MergeFrom(origin) // Set original obeject

	// Check whether patch is necessary or not
	if !reflect.DeepEqual(origin.Spec, target.Spec) {
		subResourceLogger.Info("Patch notary")
		if err := r.Patch(context.TODO(), target, originObject); err != nil {
			subResourceLogger.Error(err, "Unknown error patching")
			return err
		}
	}

	// Check whether patch is necessary or not about status
	if !reflect.DeepEqual(origin.Status, target.Status) {
		subResourceLogger.Info("Patch notary status")
		if err := r.Status().Patch(context.TODO(), target, originObject); err != nil {
			subResourceLogger.Error(err, "Unknown error patching status")
			return err
		}
	}

	return nil
}

func collectNotarySubController(serviceType regv1.NotaryServiceType) []notaryctl.NotarySubresource {
	collection := []notaryctl.NotarySubresource{}
	// [TODO] Add Subresources in here
	collection = append(collection,
		&notaryctl.NotaryDBPVC{}, &notaryctl.NotaryDBService{},
		&notaryctl.NotaryServerService{}, &notaryctl.NotarySignerService{},
		&notaryctl.NotaryServerSecret{},
		&notaryctl.NotarySignerSecret{},
		&notaryctl.NotaryDBPod{},
		&notaryctl.NotaryServerPod{},
		&notaryctl.NotarySignerPod{},
	)
	if serviceType == "Ingress" {
		collection = append(collection, &notaryctl.NotaryServerIngress{})
	}

	return collection
}
