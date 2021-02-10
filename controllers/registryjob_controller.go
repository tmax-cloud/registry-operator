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
	"github.com/tmax-cloud/registry-operator/pkg/scheduler"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
)

const (
	finalizer = "tmax.io/finalizer"
)

// RegistryJobReconciler reconciles a RegistryJob object
type RegistryJobReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Scheduler *scheduler.Scheduler
}

// +kubebuilder:rbac:groups=tmax.io,resources=registryjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=registryjobs/status,verbs=get;update;patch

// Reconcile reconciles RegistryJob
func (r *RegistryJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling RegistryJob")

	instance := &tmaxiov1.RegistryJob{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	original := instance.DeepCopy()

	exit, err := r.handleFinalizer(instance, original)
	if err != nil {
		log.Error(err, "")
		r.patchStatus(instance, original, err.Error())
		return ctrl.Result{}, nil
	}
	if exit {
		return ctrl.Result{}, nil
	}

	// Notify state change to scheduler
	defer r.Scheduler.Notify(instance)

	// Skip if it's ended
	if instance.Status.CompletionTime != nil {
		// Delete if its' ttl is 0
		if instance.Spec.TTL == 0 {
			if err := r.Client.Delete(context.Background(), instance); err != nil {
				log.Error(err, "")
			}
		}
		return ctrl.Result{}, nil
	}

	// Set initial state and exit
	if instance.Status.State == "" {
		instance.Status.State = tmaxiov1.RegistryJobStatePending
		if err := r.Client.Status().Update(context.Background(), instance); err != nil {
			log.Error(err, "")
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *RegistryJobReconciler) handleFinalizer(instance, original *tmaxiov1.RegistryJob) (bool, error) {
	// Check first if finalizer is already set
	found := false
	idx := -1
	for i, f := range instance.Finalizers {
		if f == finalizer {
			found = true
			idx = i
			break
		}
	}
	if !found {
		instance.Finalizers = append(instance.Finalizers, finalizer)
		p := client.MergeFrom(original)
		if err := r.Client.Patch(context.Background(), instance, p); err != nil {
			return false, err
		}
		return true, nil
	}

	// Deletion check-up
	if instance.DeletionTimestamp != nil && idx >= 0 {
		// Notify scheduler
		r.Scheduler.Notify(instance)

		// Delete finalizer
		if len(instance.Finalizers) == 1 {
			instance.Finalizers = nil
		} else {
			last := len(instance.Finalizers) - 1
			instance.Finalizers[idx] = instance.Finalizers[last]
			instance.Finalizers[last] = ""
			instance.Finalizers = instance.Finalizers[:last]
		}

		p := client.MergeFrom(original)
		if err := r.Client.Patch(context.Background(), instance, p); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (r *RegistryJobReconciler) patchStatus(instance, original *tmaxiov1.RegistryJob, message string) {
	instance.Status.State = tmaxiov1.RegistryJobStateFailed
	instance.Status.Message = message
	p := client.MergeFrom(original)
	if err := r.Client.Status().Patch(context.Background(), instance, p); err != nil {
		r.Log.Error(err, "")
	}
}

// collectTTLAll collects RegistryJobs who are older than TTL
func (r *RegistryJobReconciler) collectTTLAll() {
	list := &tmaxiov1.RegistryJobList{}
	if err := r.Client.List(context.Background(), list); err != nil {
		if _, ok := err.(*cache.ErrCacheNotStarted); !ok {
			log.Error(err, "")
		}
		return
	}

	// Sync all RCJs
	for _, cj := range list.Items {
		if err := r.collectTTL(&cj); err != nil {
			log.Error(err, "")
		}
	}
}

// collectTTL deletes RegistryJob if it's older than its' TTL
func (r *RegistryJobReconciler) collectTTL(j *tmaxiov1.RegistryJob) error {
	if j.Status.CompletionTime == nil || j.Spec.TTL <= 0 {
		return nil
	}

	// If now is after completion+ttl, delete it
	if time.Now().After(j.Status.CompletionTime.Time.Add(time.Duration(j.Spec.TTL) * time.Second)) {
		if err := r.Client.Delete(context.Background(), j); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the reconciler
func (r *RegistryJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.RegistryJob{}).
		Complete(r)

	// Start TTL collector
	stopChan := make(chan struct{})
	go wait.Until(r.collectTTLAll, 10, stopChan)

	return err
}
