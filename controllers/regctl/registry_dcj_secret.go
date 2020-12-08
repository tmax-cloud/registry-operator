package regctl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"

	"github.com/tmax-cloud/registry-operator/internal/utils"

	"github.com/tmax-cloud/registry-operator/internal/schemes"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	SecretDCJTypeName = regv1.ConditionTypeSecretDockerConfigJson
)

type RegistryDCJSecret struct {
	secretDCJ *corev1.Secret
	logger    *utils.RegistryLogger
}

func (r *RegistryDCJSecret) Handle(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	err := r.get(c, reg)
	if err != nil {
		if createError := r.create(c, reg, patchReg, scheme); createError != nil {
			r.logger.Error(createError, "Create failed in Handle")
			return createError
		}
	}

	if isValid := r.compare(reg); isValid == nil {
		if deleteError := r.delete(c, patchReg); deleteError != nil {
			r.logger.Error(deleteError, "Delete failed in Handle")
			return deleteError
		}
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryDCJSecret) Ready(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretDCJTypeName,
	}

	defer utils.SetCondition(err, patchReg, &condition)

	if useGet {
		if err = r.get(c, reg); err != nil {
			r.logger.Error(err, "Get failed")
			return err
		}
	}

	err = regv1.MakeRegistryError("Secret DCJ Error")
	if _, ok := r.secretDCJ.Data[schemes.DockerConfigJson]; !ok {
		r.logger.Error(err, "No certificate in data")
		return nil
	}

	condition.Status = corev1.ConditionTrue
	err = nil
	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryDCJSecret) create(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	condition := status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretDCJTypeName,
	}

	if err := controllerutil.SetControllerReference(reg, r.secretDCJ, scheme); err != nil {
		utils.SetCondition(err, patchReg, &condition)
		return err
	}

	if err := c.Create(context.TODO(), r.secretDCJ); err != nil {
		r.logger.Error(err, "Create failed")
		utils.SetCondition(err, patchReg, &condition)
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryDCJSecret) get(c client.Client, reg *regv1.Registry) error {
	r.secretDCJ = schemes.DCJSecret(reg)
	if r.secretDCJ == nil {
		return regv1.MakeRegistryError("Registry has no fields DCJ Secret required")
	}
	r.logger = utils.NewRegistryLogger(*r, r.secretDCJ.Namespace, r.secretDCJ.Name)

	req := types.NamespacedName{Name: r.secretDCJ.Name, Namespace: r.secretDCJ.Namespace}
	if err := c.Get(context.TODO(), req, r.secretDCJ); err != nil {
		r.logger.Error(err, "Get failed")
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryDCJSecret) patch(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	return nil
}

func (r *RegistryDCJSecret) delete(c client.Client, patchReg *regv1.Registry) error {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretDCJTypeName,
	}

	if err := c.Delete(context.TODO(), r.secretDCJ); err != nil {
		r.logger.Error(err, "Delete failed")
		utils.SetCondition(err, patchReg, condition)
		return err
	}

	return nil
}

func (r *RegistryDCJSecret) compare(reg *regv1.Registry) []utils.Diff {
	data := r.secretDCJ.Data
	val, ok := data[schemes.DockerConfigJson]
	if !ok {
		return nil
	}

	dockerConfig := schemes.DockerConfig{}
	json.Unmarshal(val, &dockerConfig)
	port := 443
	clusterIP := ""
	domainIP := ""
	if reg.Spec.RegistryService.ServiceType == regv1.RegServiceTypeLoadBalancer {
		port = reg.Spec.RegistryService.LoadBalancer.Port
		clusterIP = reg.Status.ClusterIP + ":" + strconv.Itoa(port)
		domainIP = reg.Status.LoadBalancerIP + ":" + strconv.Itoa(port)
	} else {
		clusterIP = reg.Status.ClusterIP + ":" + strconv.Itoa(port)
		domainIP = reg.Name + "." + reg.Spec.RegistryService.Ingress.DomainName + ":" + strconv.Itoa(port)
	}
	for key, element := range dockerConfig.Auths {
		if key != clusterIP && key != domainIP {
			return nil
		}
		loginAndPassword, _ := base64.StdEncoding.DecodeString(element.Auth)
		if string(loginAndPassword) != reg.Spec.LoginId+":"+reg.Spec.LoginPassword {
			return nil
		}
	}

	r.logger.Info("Succeed")
	return []utils.Diff{}
}