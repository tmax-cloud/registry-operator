package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/utils"

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
func NewRegistryService(client client.Client, scheme *runtime.Scheme, reg *regv1.Registry, cond status.ConditionType, logger logr.Logger) *RegistryService {
	return &RegistryService{
		c:      client,
		scheme: scheme,
		cond:   cond,
		logger: logger.WithName("Service"),
		svc:    schemes.Service(reg),
	}
}

// RegistryService things to handle service resource
type RegistryService struct {
	c      client.Client
	scheme *runtime.Scheme
	cond   status.ConditionType
	svc    *corev1.Service
	logger logr.Logger
}

// Handle makes service to be in the desired state
func (r *RegistryService) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("CreateIfNotExist")

	if err := r.get(reg); err != nil {
		r.setConditionFalseWithError(err, patchReg)
		if errors.IsNotFound(err) {
			if createError := r.create(reg, patchReg); createError != nil {
				logger.Error(createError, "Create failed in CreateIfNotExist")
				r.setConditionFalseWithError(createError, patchReg)
				return createError
			}
			logger.Info("Create Succeeded")
		} else {
			logger.Error(err, "service is error")
			return err
		}
		return nil
	}

	diff := r.compare(reg)
	if len(diff) > 0 {
		logger.Info("service must be patched")
		r.setConditionFalseWithError(nil, patchReg)
		if err := r.patch(reg, patchReg, diff); err != nil {
			r.setConditionFalseWithError(err, patchReg)
			return err
		}
	}

	logger.Info("Succeed")
	return nil
}

// Ready checks that service is ready
func (r *RegistryService) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	logger := r.logger.WithName("IsReady")
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeService,
	}
	defer utils.SetErrorConditionIfChanged(patchReg, reg, condition, err)

	if useGet {
		if err = r.get(reg); err != nil {
			logger.Error(err, "Getting Service error")
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
		logger.Info("LoadBalancer info", "LoadBalancer IP", lbIP)
	} else if r.svc.Spec.Type == corev1.ServiceTypeClusterIP {
		if r.svc.Spec.ClusterIP == "" {
			err = regv1.MakeRegistryError("NotReady")
			return err
		}
		logger.Info("Service Type is ClusterIP(Ingress)")
		// [TODO]
	}
	patchReg.Status.ClusterIP = r.svc.Spec.ClusterIP
	condition.Status = corev1.ConditionTrue
	logger.Info("Succeed")
	return nil
}

func (r *RegistryService) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("create")
	if err := controllerutil.SetControllerReference(reg, r.svc, r.scheme); err != nil {
		logger.Error(err, "Set owner reference failed")
		return err
	}

	if err := r.c.Create(context.TODO(), r.svc); err != nil {
		logger.Error(err, "Create failed")
		return err
	}

	logger.Info("Succeed")
	return nil
}

func (r *RegistryService) get(reg *regv1.Registry) error {
	logger := r.logger.WithName("get")
	req := types.NamespacedName{Name: r.svc.Name, Namespace: r.svc.Namespace}
	if err := r.c.Get(context.TODO(), req, r.svc); err != nil {
		logger.Error(err, "Get Failed")
		return err
	}
	logger.Info("Succeed")
	return nil
}

func (r *RegistryService) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	logger := r.logger.WithName("patch")
	target := r.svc.DeepCopy()
	originObject := client.MergeFrom(r.svc)

	logger.Info("Get", "Patch", fmt.Sprintf("%+v\n", diff))

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
		logger.Error(err, "Unknown error patch")
		return err
	}
	return nil
}

func (r *RegistryService) delete(patchReg *regv1.Registry) error {
	logger := r.logger.WithName("delete")
	if err := r.c.Delete(context.TODO(), r.svc); err != nil {
		logger.Error(err, "Delete failed")
		return err
	}
	logger.Info("Succeed")
	return nil
}

func (r *RegistryService) compare(reg *regv1.Registry) []utils.Diff {
	logger := r.logger.WithName("compare")
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

	logger.Info("Succeed")
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

func (r *RegistryService) setConditionFalseWithError(e error, reg *regv1.Registry) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeService,
	}
	utils.SetCondition(e, reg, condition)
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
