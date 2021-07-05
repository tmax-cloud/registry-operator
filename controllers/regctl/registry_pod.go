package regctl

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistryPod contains things to handle pod resource
type RegistryPod struct {
	c            client.Client
	manifest     func() (interface{}, error)
	cond         status.ConditionType
	requirements []status.ConditionType
	logger       logr.Logger
}

// NewRegistryPod creates new registry pod controller
// deps: deployment
func NewRegistryPod(cli client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryPod {
	return &RegistryPod{
		c:        cli,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("Pod"),
	}
}

func (r *RegistryPod) ReconcileByConditionStatus(reg *regv1.Registry) (bool, error) {
	var err error
	defer func() {
		if err != nil {
			reg.Status.Conditions.SetCondition(
				status.Condition{
					Type:    r.cond,
					Status:  corev1.ConditionFalse,
					Message: err.Error(),
				})
		}
	}()

	for _, dep := range r.requirements {
		if !reg.Status.Conditions.GetCondition(dep).IsTrue() {
			r.logger.Info(string(r.cond) + " needs " + string(dep))
			return true, nil
		}
	}

	ctx := context.TODO()
	podList := &corev1.PodList{}
	if err = r.c.List(ctx, podList, &client.ListOptions{
		Namespace: reg.Namespace,
		LabelSelector: labels.SelectorFromSet(labels.Set(map[string]string{
			"app":  "registry",
			"apps": schemes.SubresourceName(reg, schemes.SubTypeRegistryDeployment),
		})),
	}); err != nil {
		return false, err
	}

	if len(podList.Items) == 0 || podList.Items[0].Status.Phase != "Running" {
		r.logger.Info("pod not ready")
		return true, nil
	}

	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})

	return false, nil
}

func (r *RegistryPod) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
