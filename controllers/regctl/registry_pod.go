package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRegistryPod creates new registry pod controller
// deps: deployment
func NewRegistryPod(client client.Client, scheme *runtime.Scheme, reg *regv1.Registry, cond status.ConditionType, logger logr.Logger, deps ...Dependent) *RegistryPod {
	return &RegistryPod{
		c:      client,
		scheme: scheme,
		cond:   cond,
		logger: logger.WithName("Pod"),
		deps:   deps,
	}
}

// RegistryPod contains things to handle pod resource
type RegistryPod struct {
	c      client.Client
	scheme *runtime.Scheme
	cond   status.ConditionType
	deps   []Dependent
	pod    *corev1.Pod
	logger logr.Logger
}

// Handle makes pod to be in the desired state
func (r *RegistryPod) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("CreateIfNotExist")
	for _, dep := range r.deps {
		if !dep.IsSuccessfullyCompleted(reg) {
			err := fmt.Errorf("unable to handle %s: %s condition is not satisfied", r.Condition(), dep.Condition())
			return err
		}
	}

	if err := r.get(reg); err != nil {
		logger.Error(err, "Pod error")
		r.notReady(patchReg, err)
		return err
	}

	logger.Info("Check if recreating pod is required.")
	if reg.Status.PodRecreateRequired || r.compare(reg) == nil {
		r.notReady(patchReg, nil)
		if err := r.delete(patchReg); err != nil {
			return err
		}

		logger.Info("Recreate pod.")
		patchReg.Status.PodRecreateRequired = false
	}

	return nil
}

// Ready checks that pod is ready
func (r *RegistryPod) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	logger := r.logger.WithName("IsReady")
	var err error = nil
	podCondition := &status.Condition{
		Type:   regv1.ConditionTypePod,
		Status: corev1.ConditionFalse,
	}
	contCondition := &status.Condition{
		Type:   regv1.ConditionTypeContainer,
		Status: corev1.ConditionFalse,
	}

	defer utils.SetErrorConditionIfChanged(patchReg, reg, podCondition, err)
	defer utils.SetErrorConditionIfChanged(patchReg, reg, contCondition, err)

	if r.pod == nil || useGet {
		err = r.get(reg)
		if err != nil {
			logger.Error(err, "Pod error")
			return err
		}
	}

	if r.pod == nil {
		logger.Info("Pod is nil")
		podCondition.Status = corev1.ConditionFalse
		contCondition.Status = corev1.ConditionFalse
		err = regv1.MakeRegistryError(regv1.PodNotFound)
		return err
	}

	contStatuses := r.pod.Status.ContainerStatuses
	if len(contStatuses) == 0 {
		logger.Info("Container's status is nil")
		podCondition.Status = corev1.ConditionFalse
		contCondition.Status = corev1.ConditionFalse
		err = regv1.MakeRegistryError(regv1.ContainerStatusIsNil)
		return err
	}
	contState := r.pod.Status.ContainerStatuses[0]
	var reason string

	if contState.State.Waiting != nil {
		reason = contState.State.Waiting.Reason
		logger.Info(reason)
	} else if contState.State.Running != nil {
		// logger.Info(contState.String())
		if contState.Ready {
			reason = "Running"
		} else {
			reason = "NotReady"
		}
	} else if contState.State.Terminated != nil {
		reason = contState.State.Terminated.Reason
		logger.Info(reason)
	} else {
		reason = "Unknown"
	}

	logger.Info("Get container state", "reason", reason)

	switch reason {
	case "NotReady":
		podCondition.Status = corev1.ConditionTrue
		contCondition.Status = corev1.ConditionFalse
		err = regv1.MakeRegistryError(regv1.PodNotRunning)
		return err

	case "Running":
		podCondition.Status = corev1.ConditionTrue
		contCondition.Status = corev1.ConditionTrue

	default:
		podCondition.Status = corev1.ConditionFalse
		contCondition.Status = corev1.ConditionFalse
		err = regv1.MakeRegistryError(regv1.PodNotRunning)
		return err
	}

	return nil
}

func (r *RegistryPod) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	return nil
}

func (r *RegistryPod) get(reg *regv1.Registry) error {
	logger := r.logger.WithName("get")
	r.pod = &corev1.Pod{}

	podList := &corev1.PodList{}
	label := map[string]string{}
	label["app"] = "registry"
	label["apps"] = schemes.SubresourceName(reg, schemes.SubTypeRegistryDeployment)

	labelSelector := labels.SelectorFromSet(labels.Set(label))
	listOps := &client.ListOptions{
		Namespace:     reg.Namespace,
		LabelSelector: labelSelector,
	}
	err := r.c.List(context.TODO(), podList, listOps)
	if err != nil {
		logger.Error(err, "Failed to list pods.")
		return err
	}

	if len(podList.Items) == 0 {
		return regv1.MakeRegistryError(regv1.PodNotFound)
	}

	r.pod = &podList.Items[0]

	return nil
}

func (r *RegistryPod) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	return nil
}

func (r *RegistryPod) delete(patchReg *regv1.Registry) error {
	logger := r.logger.WithName("delete")
	if err := r.c.Delete(context.TODO(), r.pod); err != nil {
		logger.Error(err, "Unknown error delete pod")
		return err
	}

	return nil
}

func (r *RegistryPod) compare(reg *regv1.Registry) []utils.Diff {
	cond1 := reg.Status.Conditions.GetCondition(regv1.ConditionTypePod)
	cond2 := reg.Status.Conditions.GetCondition(regv1.ConditionTypeContainer)

	for _, dep := range r.deps {
		for _, depTime := range dep.ModifiedTime(reg) {
			if depTime.After(cond1.LastTransitionTime.Time) ||
				depTime.After(cond2.LastTransitionTime.Time) {
				return nil
			}
		}
	}

	return []utils.Diff{}
}

// IsSuccessfullyCompleted returns true if conditions are satisfied
func (r *RegistryPod) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond1 := reg.Status.Conditions.GetCondition(regv1.ConditionTypePod)
	if cond1 == nil {
		return false
	}

	cond2 := reg.Status.Conditions.GetCondition(regv1.ConditionTypeContainer)
	if cond2 == nil {
		return false
	}

	return cond1.IsTrue() && cond2.IsTrue()
}

func (r *RegistryPod) notReady(patchReg *regv1.Registry, err error) {
	podCondition := &status.Condition{
		Type:   regv1.ConditionTypePod,
		Status: corev1.ConditionFalse,
	}

	contCondition := &status.Condition{
		Type:   regv1.ConditionTypeContainer,
		Status: corev1.ConditionFalse,
	}

	utils.SetCondition(err, patchReg, podCondition)
	utils.SetCondition(err, patchReg, contCondition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryPod) Condition() string {
	return fmt.Sprintf("%s and %s", string(regv1.ConditionTypePod), string(regv1.ConditionTypeContainer))
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryPod) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond1 := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypePod)
	if cond1 == nil {
		return nil
	}
	cond2 := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeContainer)
	if cond2 == nil {
		return nil
	}

	return []time.Time{cond1.LastTransitionTime.Time, cond2.LastTransitionTime.Time}
}
