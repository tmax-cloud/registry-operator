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
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
)

// RepositoryReconciler reconciles a Repository object
type RepositoryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=repositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=repositories/status,verbs=get;update;patch

func (r *RepositoryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("repository", req.NamespacedName)

	// If this repository is external registry's repository, do not handle deleting images
	if strings.HasPrefix(req.Name, schemes.ExternalRegistryPrefix) {
		return reconcile.Result{}, nil
	}

	// Fetch the Repository instance
	repo := &regv1.Repository{}
	if err := r.Get(context.TODO(), req.NamespacedName, repo); err != nil {
		if errors.IsNotFound(err) {
			reg, err := getRegistryByRequest(r.Client, req)
			if err != nil {
				r.Log.Error(err, "")
				return reconcile.Result{}, err
			}
			r.Log.Info("get_registry", "namespace", reg.Namespace, "name", reg.Name)

			repoName, _ := splitRepoCRName(req.Name)
			r.Log.Info("repository", "name", repoName)

			if err := sweepRegistryRepo(r.Client, reg, repoName); err != nil {
				r.Log.Error(err, "")
				return reconcile.Result{}, nil
			}

			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	reg, err := getRegistryByRequest(r.Client, req)
	if err != nil {
		r.Log.Error(err, "")
		return reconcile.Result{}, nil
	}

	if err := sweepImages(r.Client, reg, repo); err != nil {
		r.Log.Error(err, "")
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&regv1.Repository{}).
		Complete(r)
}

func getRegistryByRequest(c client.Client, request reconcile.Request) (*regv1.Registry, error) {
	registry := &regv1.Registry{}
	_, name := splitRepoCRName(request.Name)
	namespace := request.Namespace

	err := c.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, registry)
	if err != nil {
		return nil, err
	}

	return registry, nil
}

// registry name must not contain dot(`.`) character
func splitRepoCRName(crName string) (repoName, regName string) {
	parts := strings.Split(crName, ".")

	repoName = strings.Join(parts[:len(parts)-1], ".")
	regName = parts[len(parts)-1]

	return
}
