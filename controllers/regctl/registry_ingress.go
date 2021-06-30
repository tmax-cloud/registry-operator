package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/utils"

	"github.com/tmax-cloud/registry-operator/internal/schemes"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RegistryIngress contains things to handle ingress resource
type RegistryIngress struct {
	c       client.Client
	scheme  *runtime.Scheme
	cond    status.ConditionType
	deps    []Dependent
	ingress *v1beta1.Ingress
	logger  logr.Logger
}

// NewRegistryIngress creates new registry ingress controller
// deps: cert
func NewRegistryIngress(client client.Client, scheme *runtime.Scheme, reg *regv1.Registry, cond status.ConditionType, logger logr.Logger, deps ...Dependent) *RegistryIngress {
	ingress := schemes.Ingress(reg)
	if ingress == nil {
		logger.Error(fmt.Errorf("no ingress required"), "failed to initiate ingress controller")
		//return nil
	}

	return &RegistryIngress{
		c:       client,
		scheme:  scheme,
		cond:    cond,
		logger:  logger.WithName("Ingress"),
		ingress: ingress,
		deps:    deps,
	}
}

func (r *RegistryIngress) mustCreated(reg *regv1.Registry) bool {
	return reg.Status.Conditions.GetCondition(regv1.ConditionTypeIngress) != nil
}

// Handle makes ingress to be in the desired state
func (r *RegistryIngress) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("CreateIfNotExist")
	if !r.mustCreated(reg) {
		if err := r.get(reg); err != nil {
			return nil
		}
		if err := r.delete(reg); err != nil {
			logger.Error(err, "failed to delete ingress")
			return err
		}
		return nil
	}

	for _, dep := range r.deps {
		if !dep.IsSuccessfullyCompleted(reg) {
			err := fmt.Errorf("unable to handle %s: %s condition is not satisfied", r.Condition(), dep.Condition())
			r.notReady(patchReg, err)
			return err
		}
	}

	if err := r.get(reg); err != nil {
		// if r.ingress == nil {
		// 	r.setConditionFalseWithError(patchReg, err)
		// 	return err
		// }

		if errors.IsNotFound(err) {
			r.notReady(patchReg, err)
			if createError := r.create(reg, patchReg); createError != nil {
				logger.Error(createError, "Create failed in CreateIfNotExist")
				r.notReady(patchReg, createError)
				return createError
			}
			logger.Info("Create Succeeded")
		} else {
			logger.Error(err, "ingress is error")
			return err
		}
		return nil
	}

	if isValid := r.compare(reg); isValid == nil {
		r.notReady(patchReg, nil)
		if err := r.delete(patchReg); err != nil {
			logger.Error(err, "Delete failed in CreateIfNotExist")
			r.notReady(patchReg, nil)
			return err
		}
	}

	logger.Info("Succeed")
	return nil
}

// Ready checks that ingress is ready
func (r *RegistryIngress) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	logger := r.logger.WithName("IsReady")
	if !r.mustCreated(reg) {
		return nil
	}

	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeIngress,
	}

	defer utils.SetErrorConditionIfChanged(patchReg, reg, condition, err)
	if useGet {
		if err = r.get(reg); err != nil {
			logger.Error(err, "Get failed")
			return err
		}
	}

	if _, ok := r.ingress.Annotations["kubernetes.io/ingress.class"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if _, ok := r.ingress.Annotations["nginx.ingress.kubernetes.io/proxy-connect-timeout"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if _, ok := r.ingress.Annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if _, ok := r.ingress.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if val, ok := r.ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"]; ok {
		if val != "HTTPS" {
			err = regv1.MakeRegistryError("Ingress Error")
			return err
		}
	} else {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if _, ok := r.ingress.Annotations["nginx.ingress.kubernetes.io/proxy-body-size"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}

	if len(r.ingress.Spec.TLS) > 0 {
		for _, host := range r.ingress.Spec.TLS[0].Hosts {
			patchReg.Status.ServerURL = "https://" + host
		}
	}

	condition.Status = corev1.ConditionTrue
	logger.Info("Succeed")
	return nil
}

func (r *RegistryIngress) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("create")
	r.ingress = schemes.Ingress(reg)
	if r.ingress == nil {
		return regv1.MakeRegistryError("Registry has no fields Ingress required")
	}

	if err := controllerutil.SetControllerReference(reg, r.ingress, r.scheme); err != nil {
		logger.Error(err, "Controller reference failed")
		return err
	}

	if err := r.c.Create(context.TODO(), r.ingress); err != nil {
		logger.Error(err, "Create failed")
		return err
	}

	logger.Info("Succeed")
	return nil
}

func (r *RegistryIngress) get(reg *regv1.Registry) error {
	logger := r.logger.WithName("get")
	req := types.NamespacedName{Name: r.ingress.Name, Namespace: r.ingress.Namespace}
	if err := r.c.Get(context.TODO(), req, r.ingress); err != nil {
		logger.Error(err, "Get failed")
		return err
	}

	logger.Info("Succeed")
	return nil
}

func (r *RegistryIngress) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	return nil
}

func (r *RegistryIngress) delete(patchReg *regv1.Registry) error {
	logger := r.logger.WithName("delete")
	if err := r.c.Delete(context.TODO(), r.ingress); err != nil {
		logger.Error(err, "Delete failed")
		return err
	}

	return nil
}

func (r *RegistryIngress) compare(reg *regv1.Registry) []utils.Diff {
	logger := r.logger.WithName("compare")
	diff := []utils.Diff{}

	if reg.Spec.RegistryService.ServiceType != "Ingress" {
		return nil
	}
	registryDomain := schemes.RegistryDomainName(reg)

	for _, ingressTLS := range r.ingress.Spec.TLS {
		for _, host := range ingressTLS.Hosts {
			if host != registryDomain {
				return nil
			}
		}
	}

	for _, ingressRule := range r.ingress.Spec.Rules {
		if ingressRule.Host != registryDomain {
			return nil
		}
	}

	logger.Info("Succeed")
	return diff
}

// IsSuccessfullyCompleted returns true if condition is satisfied
func (r *RegistryIngress) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond := reg.Status.Conditions.GetCondition(regv1.ConditionTypeIngress)
	if cond == nil {
		return false
	}

	return cond.IsTrue()
}

func (r *RegistryIngress) notReady(patchReg *regv1.Registry, err error) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeIngress,
	}
	utils.SetCondition(err, patchReg, condition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryIngress) Condition() string {
	return string(regv1.ConditionTypeIngress)
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryIngress) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeIngress)
	if cond == nil {
		return nil
	}

	return []time.Time{cond.LastTransitionTime.Time}
}
