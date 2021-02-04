package regctl

import (
	"context"
	"encoding/base64"
	"encoding/json"

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

// RegistryDCJSecret contains things to handle docker config json secret resource
type RegistryDCJSecret struct {
	secretDCJ *corev1.Secret
	logger    *utils.RegistryLogger
}

// Handle makes docker config json secret to be in the desired state
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

// Ready checks that docker config json secret is ready
func (r *RegistryDCJSecret) Ready(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretDockerConfigJSON,
	}

	defer utils.SetCondition(err, patchReg, &condition)

	if useGet {
		if err = r.get(c, reg); err != nil {
			r.logger.Error(err, "Get failed")
			return err
		}
	}

	if _, ok := r.secretDCJ.Data[schemes.DockerConfigJson]; !ok {
		err = regv1.MakeRegistryError("Secret DCJ Error")
		r.logger.Error(err, "No certificate in data")
		return err
	}

	condition.Status = corev1.ConditionTrue
	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryDCJSecret) create(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	condition := status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretDockerConfigJSON,
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
	r.logger = utils.NewRegistryLogger(*r, reg.Namespace, schemes.SubresourceName(reg, schemes.SubTypeRegistryDCJSecret))
	r.secretDCJ = schemes.DCJSecret(reg)
	if r.secretDCJ == nil {
		return regv1.MakeRegistryError("Registry has no fields DCJ Secret required")
	}

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
		Type:   regv1.ConditionTypeSecretDockerConfigJSON,
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
		if string(loginAndPassword) != reg.Spec.LoginId+":"+reg.Spec.LoginPassword {
			return nil
		}
	}

	r.logger.Info("Succeed")
	return []utils.Diff{}
}
