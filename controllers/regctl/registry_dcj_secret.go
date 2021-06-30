package regctl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/utils"

	"github.com/tmax-cloud/registry-operator/internal/schemes"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RegistryDCJSecret contains things to handle docker config json secret resource
type RegistryDCJSecret struct {
	c         client.Client
	scheme    *runtime.Scheme
	cond      status.ConditionType
	deps      []Dependent
	secretDCJ *corev1.Secret
	logger    logr.Logger
}

// NewRegistryDCJSecret creates new registry docker config json secret controller
// deps: service
func NewRegistryDCJSecret(client client.Client, scheme *runtime.Scheme, cond status.ConditionType, logger logr.Logger, deps ...Dependent) *RegistryDCJSecret {
	return &RegistryDCJSecret{
		c:      client,
		scheme: scheme,
		cond:   cond,
		logger: logger.WithName("DockerConfigSecret"),
		deps:   deps,
	}
}

// Handle makes docker config json secret to be in the desired state
func (r *RegistryDCJSecret) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("CreateIfNotExist")
	for _, dep := range r.deps {
		if !dep.IsSuccessfullyCompleted(reg) {
			err := fmt.Errorf("unable to handle %s: %s condition is not satisfied", r.Condition(), dep.Condition())
			return err
		}
	}

	if err := r.get(reg); err != nil {
		r.notReady(patchReg, err)
		if errors.IsNotFound(err) {
			if createError := r.create(reg, patchReg); createError != nil {
				logger.Error(createError, "Create failed in CreateIfNotExist")
				r.notReady(patchReg, createError)
				return createError
			}
			logger.Info("Create Succeeded")
		} else {
			logger.Error(err, "docker config json secret error")
			return err
		}
		return nil
	}

	if isValid := r.compare(reg); isValid == nil {
		r.notReady(patchReg, nil)
		if deleteError := r.delete(patchReg); deleteError != nil {
			logger.Error(deleteError, "Delete failed in CreateIfNotExist")
			r.notReady(patchReg, deleteError)
			return deleteError
		}
	}

	logger.Info("Succeed")
	return nil
}

// Ready checks that docker config json secret is ready
func (r *RegistryDCJSecret) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	logger := r.logger.WithName("IsReady")
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretDockerConfigJSON,
	}

	defer utils.SetErrorConditionIfChanged(patchReg, reg, condition, err)

	if useGet {
		if err = r.get(reg); err != nil {
			logger.Error(err, "Get failed")
			return err
		}
	}

	if _, ok := r.secretDCJ.Data[schemes.DockerConfigJson]; !ok {
		err = regv1.MakeRegistryError("Secret DCJ Error")
		logger.Error(err, "No certificate in data")
		return err
	}

	condition.Status = corev1.ConditionTrue
	logger.Info("Succeed")
	return nil
}

func (r *RegistryDCJSecret) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("create")
	condition := status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretDockerConfigJSON,
	}

	if err := controllerutil.SetControllerReference(reg, r.secretDCJ, r.scheme); err != nil {
		utils.SetCondition(err, patchReg, &condition)
		return err
	}

	if err := r.c.Create(context.TODO(), r.secretDCJ); err != nil {
		logger.Error(err, "Create failed")
		utils.SetCondition(err, patchReg, &condition)
		return err
	}

	logger.Info("Succeed")
	return nil
}

func (r *RegistryDCJSecret) get(reg *regv1.Registry) error {
	logger := r.logger.WithName("get")
	r.secretDCJ = schemes.DCJSecret(reg)
	if r.secretDCJ == nil {
		return regv1.MakeRegistryError("Registry has no fields DCJ Secret required")
	}

	req := types.NamespacedName{Name: r.secretDCJ.Name, Namespace: r.secretDCJ.Namespace}
	if err := r.c.Get(context.TODO(), req, r.secretDCJ); err != nil {
		logger.Error(err, "Get failed")
		return err
	}

	logger.Info("Succeed")
	return nil
}

func (r *RegistryDCJSecret) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	return nil
}

func (r *RegistryDCJSecret) delete(patchReg *regv1.Registry) error {
	logger := r.logger.WithName("delete")
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretDockerConfigJSON,
	}

	if err := r.c.Delete(context.TODO(), r.secretDCJ); err != nil {
		logger.Error(err, "Delete failed")
		utils.SetCondition(err, patchReg, condition)
		return err
	}

	return nil
}

func (r *RegistryDCJSecret) compare(reg *regv1.Registry) []utils.Diff {
	logger := r.logger.WithName("compare")
	diff := []utils.Diff{}
	cond := reg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretDockerConfigJSON)

	for _, dep := range r.deps {
		for _, depTime := range dep.ModifiedTime(reg) {
			if depTime.After(cond.LastTransitionTime.Time) {
				return nil
			}
		}
	}

	data := r.secretDCJ.Data
	val, ok := data[schemes.DockerConfigJson]
	if !ok {
		return nil
	}

	dockerConfig := schemes.DockerConfig{}
	if err := json.Unmarshal(val, &dockerConfig); err != nil {
		return nil
	}
	clusterIP := ""
	domainIP := ""
	if reg.Spec.RegistryService.ServiceType == regv1.RegServiceTypeLoadBalancer {
		// port = reg.Spec.RegistryService.LoadBalancer.Port
		clusterIP = reg.Status.ClusterIP
		domainIP = reg.Status.LoadBalancerIP
	} else {
		clusterIP = reg.Status.ClusterIP
		domainIP = schemes.RegistryDomainName(reg)
	}
	for key, element := range dockerConfig.Auths {
		if key != clusterIP && key != domainIP {
			return nil
		}
		loginAndPassword, _ := base64.StdEncoding.DecodeString(element.Auth)
		if string(loginAndPassword) != reg.Spec.LoginID+":"+reg.Spec.LoginPassword {
			return nil
		}
	}

	logger.Info("Succeed")
	return diff
}

// IsSuccessfullyCompleted returns true if condition is satisfied
func (r *RegistryDCJSecret) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond := reg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretDockerConfigJSON)
	if cond == nil {
		return false
	}

	return cond.IsTrue()
}

func (r *RegistryDCJSecret) notReady(patchReg *regv1.Registry, err error) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretDockerConfigJSON,
	}
	utils.SetCondition(err, patchReg, condition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryDCJSecret) Condition() string {
	return string(regv1.ConditionTypeSecretDockerConfigJSON)
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryDCJSecret) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretDockerConfigJSON)
	if cond == nil {
		return nil
	}

	return []time.Time{cond.LastTransitionTime.Time}
}
