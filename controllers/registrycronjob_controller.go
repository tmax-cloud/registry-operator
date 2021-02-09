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
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron"
	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RegistryCronJobController reconciles a RegistryJob object
type RegistryCronJobController struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	stopChan chan struct{}
}

// StartRegistryCronJobController creates a RegistryCronJobController and starts it
func StartRegistryCronJobController(c client.Client, log logr.Logger, scheme *runtime.Scheme) *RegistryCronJobController {
	rcjc := &RegistryCronJobController{
		Client:   c,
		Log:      log,
		Scheme:   scheme,
		stopChan: make(chan struct{}),
	}
	go wait.Until(rcjc.syncAll, 10*time.Second, rcjc.stopChan)
	return rcjc
}

// +kubebuilder:rbac:groups=tmax.io,resources=registrycronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=registrycronjobs/status,verbs=get;update;patch

func (r *RegistryCronJobController) syncAll() {
	list := &v1.RegistryCronJobList{}
	if err := r.Client.List(context.Background(), list); err != nil {
		if _, ok := err.(*cache.ErrCacheNotStarted); !ok {
			log.Error(err, "")
		}
		return
	}

	// Sync all RCJs
	for _, cj := range list.Items {
		r.sync(&cj)
	}
}

func (r *RegistryCronJobController) sync(cj *v1.RegistryCronJob) {
	t, err := getRecentScheduleTime(cj, time.Now())
	if err != nil {
		r.Log.Error(err, "")
		return
	}

	if t == nil {
		return
	}

	// Schedule the RegistryJob
	// Create
	j := jobFromSpec(cj, *t)
	if err := controllerutil.SetOwnerReference(cj, j, r.Scheme); err != nil {
		r.Log.Error(err, "")
		return
	}
	if err := r.Client.Create(context.Background(), j); err != nil {
		r.Log.Error(err, "")
		return
	}

	// Patch RegistryCronJob's lastScheduledTime
	original := cj.DeepCopy()
	cj.Status.LastScheduledTime = &metav1.Time{Time: time.Now()}

	if err := r.Status().Patch(context.Background(), cj, client.MergeFrom(original)); err != nil {
		r.Log.Error(err, "")
	}
}

func getRecentScheduleTime(cj *v1.RegistryCronJob, now time.Time) (*time.Time, error) {
	// Last time
	lastSchedule := cj.Status.LastScheduledTime.DeepCopy()
	if lastSchedule == nil {
		lastSchedule = cj.CreationTimestamp.DeepCopy()
	}

	if lastSchedule == nil {
		return nil, fmt.Errorf("no last scheduled time")
	}

	// Parse cron spec
	schedule, err := cron.ParseStandard(cj.Spec.Schedule)
	if err != nil {
		return nil, err
	}

	earliestNext := schedule.Next(lastSchedule.Time)
	if earliestNext.After(now) {
		return nil, nil
	}

	var schedules []time.Time
	var recentSchedule time.Time
	for recentSchedule = earliestNext; !recentSchedule.After(now); recentSchedule = schedule.Next(recentSchedule) {
		schedules = append(schedules, recentSchedule)
		if len(schedules) > 100 {
			return nil, fmt.Errorf("too many unstarted schedules")
		}
	}

	return &recentSchedule, nil
}

func jobFromSpec(cj *v1.RegistryCronJob, t time.Time) *v1.RegistryJob {
	rj := &v1.RegistryJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", cj.Name, t.Unix()),
			Namespace: cj.Namespace,
		},
	}
	cj.Spec.JobSpec.DeepCopyInto(&rj.Spec)
	return rj
}
