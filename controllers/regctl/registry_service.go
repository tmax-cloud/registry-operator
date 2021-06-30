package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/utils"

	"github.com/tmax-cloud/registry-operator/internal/schemes"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	serviceTypeDiffKey = "ServiceType"
)

// NewRegistryService creates new registry service
func NewRegistryService(client client.Client, scheme *runtime.Scheme, reg *regv1.Registry, logger logr.Logger) *RegistryService {
	return &RegistryService{
		c:      client,
		scheme: scheme,
		reg:    reg,
		logger: logger.WithName("Service"),
	}
}

// RegistryService things to handle service resource
type RegistryService struct {
	c      client.Client
	scheme *runtime.Scheme
	reg    *regv1.Registry
	svc    *corev1.Service
	logger logr.Logger
}

// Handle makes service to be in the desired state
func (r *RegistryService) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	if err := r.get(reg); err != nil {
		r.notReady(patchReg, err)
		if errors.IsNotFound(err) {
			if createError := r.create(reg, patchReg); createError != nil {
				r.logger.Error(createError, "Create failed in CreateIfNotExist")
				r.notReady(patchReg, createError)
				return createError
			}
			r.logger.Info("Create Succeeded")
		} else {
			r.logger.Error(err, "service is error")
			return err
		}
		return nil
	}

	diff := r.compare(reg)
	if len(diff) > 0 {
		r.logger.Info("service must be patched")
		r.notReady(patchReg, nil)
		if err := r.patch(reg, patchReg, diff); err != nil {
			r.notReady(patchReg, err)
			return err
		}
	}

	r.logger.Info("Succeed")
	return nil
}

// Ready checks that service is ready
func (r *RegistryService) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeService,
	}
	defer utils.SetErrorConditionIfChanged(patchReg, reg, condition, err)

	if useGet {
		if err = r.get(reg); err != nil {
			r.logger.Error(err, "Getting Service error")
			return err
		}
	}

	if r.svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		loadBalancer := r.svc.Status.LoadBalancer
		lbIP := ""
		// [TODO] Specific Condition is needed
		if len(loadBalancer.Ingress) == 1 {
			if loadBalancer.Ingress[0].Hostname == "" {
				lbIP = loadBalancer.Ingress[0].IP
			} else {
				lbIP = loadBalancer.Ingress[0].Hostname
			}
		} else if len(loadBalancer.Ingress) == 0 {
			// Several Ingress
			err = regv1.MakeRegistryError("NotReady")
			return err
		}
		patchReg.Status.LoadBalancerIP = lbIP
		patchReg.Status.ServerURL = "https://" + lbIP
		r.logger.Info("LoadBalancer info", "LoadBalancer IP", lbIP)
	} else if r.svc.Spec.Type == corev1.ServiceTypeClusterIP {
		if r.svc.Spec.ClusterIP == "" {
			err = regv1.MakeRegistryError("NotReady")
			return err
		}
		r.logger.Info("Service Type is ClusterIP(Ingress)")
		// [TODO]
	}
	patchReg.Status.ClusterIP = r.svc.Spec.ClusterIP
	condition.Status = corev1.ConditionTrue
	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryService) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	if err := controllerutil.SetControllerReference(reg, r.svc, r.scheme); err != nil {
		r.logger.Error(err, "Set owner reference failed")
		return err
	}

	if err := r.c.Create(context.TODO(), r.svc); err != nil {
		r.logger.Error(err, "Create failed")
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryService) get(reg *regv1.Registry) error {
	if r.svc == nil {
		r.svc = schemes.Service(reg)
	}

	req := types.NamespacedName{Name: r.svc.Name, Namespace: r.svc.Namespace}
	if err := r.c.Get(context.TODO(), req, r.svc); err != nil {
		r.logger.Error(err, "Get Failed")
		return err
	}
	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryService) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	target := r.svc.DeepCopy()
	originObject := client.MergeFrom(r.svc)

	r.logger.Info("Get", "Patch", fmt.Sprintf("%+v\n", diff))

	for _, d := range diff {
		switch d.Key {
		case serviceTypeDiffKey:
			switch d.Type {
			case utils.Replace:
				target.Spec.Type = d.Value.(corev1.ServiceType)
			}
		}
	}

	// Patch
	if err := r.c.Patch(context.TODO(), target, originObject); err != nil {
		r.logger.Error(err, "Unknown error patch")
		return err
	}
	return nil
}

func (r *RegistryService) delete(patchReg *regv1.Registry) error {
	if err := r.c.Delete(context.TODO(), r.svc); err != nil {
		r.logger.Error(err, "Delete failed")
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryService) compare(reg *regv1.Registry) []utils.Diff {
	diff := []utils.Diff{}
	switch reg.Spec.RegistryService.ServiceType {
	case "LoadBalancer":
		if r.svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			diff = append(diff, utils.Diff{Type: utils.Replace, Key: serviceTypeDiffKey, Value: corev1.ServiceTypeLoadBalancer})
		}
	case "Ingress":
		if r.svc.Spec.Type != corev1.ServiceTypeClusterIP {
			diff = append(diff, utils.Diff{Type: utils.Replace, Key: serviceTypeDiffKey, Value: corev1.ServiceTypeClusterIP})
		}
	}

	r.logger.Info("Succeed")
	return diff
}

// IsSuccessfullyCompleted returns true if condition is satisfied
func (r *RegistryService) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond := reg.Status.Conditions.GetCondition(regv1.ConditionTypeService)
	if cond == nil {
		return false
	}

	return cond.IsTrue()
}

func (r *RegistryService) notReady(patchReg *regv1.Registry, err error) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeService,
	}
	utils.SetCondition(err, patchReg, condition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryService) Condition() string {
	return string(regv1.ConditionTypeService)
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryService) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeService)
	if cond == nil {
		return nil
	}

	return []time.Time{cond.LastTransitionTime.Time}
}
