package regctl

import (
	"context"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-lib/status"
	"sigs.k8s.io/controller-runtime/pkg/client"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// UpdateRegistryStatus ...
// If registry status is updated, return true.
func UpdateRegistryStatus(c client.Client, reg *regv1.Registry) (bool, error) {
	reqLogger := logf.Log.WithName("controller_registry").WithValues("Request.Namespace", reg.Namespace, "Request.Name", reg.Name)
	falseTypes := []status.ConditionType{}
	checkTypes := getCheckTypes(reg)

	var desiredStatus regv1.Status

	if len(reg.Status.Conditions) != len(checkTypes) {
		if err := initRegistryStatus(c, reg); err != nil {
			return false, err
		}
		return true, nil
	}

	// Check if all subresources are true
	reqLogger.Info("Check if status fields are normal.")
	for _, t := range checkTypes {
		if reg.Status.Conditions.IsUnknownFor(t) {
			reqLogger.Info("Initialize status fields")
			if err := initRegistryStatus(c, reg); err != nil {
				return false, err
			}
			return true, nil

		} else if reg.Status.Conditions.IsFalseFor(t) {
			falseTypes = append(falseTypes, t)
		}
	}

	reqLogger.Info("Get desired status.")
	for _, t := range falseTypes {
		reqLogger.Info("false_type", "false", t)
	}

	if len(falseTypes) > 1 {
		desiredStatus = regv1.StatusCreating
	} else if len(falseTypes) == 1 {
		if falseTypes[0] == regv1.ConditionTypeContainer {
			desiredStatus = regv1.StatusNotReady
		} else {
			desiredStatus = regv1.StatusCreating
		}
	} else {
		desiredStatus = regv1.StatusRunning
	}

	reqLogger.Info("desiredStatus", "status", desiredStatus)

	// Chcck if current status is desired status. If does not same, update the status.
	reqLogger.Info("Check if current status is desired status.")
	if reg.Status.Phase == string(desiredStatus) {
		return false, nil
	}
	reqLogger.Info("Current Status(" + reg.Status.Phase + ") -> Desired Status(" + string(desiredStatus) + ")")

	var message, reason string
	target := reg.DeepCopy()

	switch desiredStatus {
	case regv1.StatusCreating:
		message = "Registry is creating. All resources in registry has not yet been created."
		reason = "AllConditionsNotTrue"
	case regv1.StatusNotReady:
		message = "Registry is not ready."
		reason = "NotReady"
	case regv1.StatusRunning:
		message = "Registry is running. All registry resources are operating normally."
		reason = "Running"
	}

	target.Status.Message = message
	target.Status.Reason = reason
	target.Status.Phase = string(desiredStatus)
	target.Status.PhaseChangedAt = metav1.Now()

	// Patch the status to desired status.
	reqLogger.Info("Status update.")
	if err := c.Status().Update(context.TODO(), target); err != nil {
		reqLogger.Error(err, "failed to update status")
		return false, err
	}

	return true, nil
}

func initRegistryStatus(c client.Client, reg *regv1.Registry) error {
	reqLogger := logf.Log.WithName("controller_registry").WithValues("Request.Namespace", reg.Namespace, "Request.Name", reg.Name)

	if reg.Status.Conditions == nil {
		reg.Status.Conditions = status.NewConditions()
	}

	// Set Conditions
	checkTypes := getCheckTypes(reg)
	for _, t := range checkTypes {
		if reg.Status.Conditions.GetCondition(t) == nil {
			reqLogger.Info("New Condition: " + string(t))
			newCondition := status.Condition{Type: t, Status: corev1.ConditionFalse}
			reg.Status.Conditions.SetCondition(newCondition)
		}
	}

	for _, t := range reg.Status.Conditions {
		if !contains(checkTypes, t.Type) {
			reg.Status.Conditions.RemoveCondition(t.Type)
		}
	}

	reg.Status.Message = "registry is creating. All resources in registry has not yet been created."
	reg.Status.Reason = "AllConditionsNotTrue"
	reg.Status.Phase = string(regv1.StatusCreating)
	reg.Status.PhaseChangedAt = metav1.Now()

	err := c.Status().Update(context.TODO(), reg)
	if err != nil {
		reqLogger.Error(err, "cannot update status")
		return err
	}

	return nil
}

func getCheckTypes(reg *regv1.Registry) []status.ConditionType {
	checkTypes := []status.ConditionType{
		regv1.ConditionTypeDeployment,
		regv1.ConditionTypePod,
		regv1.ConditionTypeContainer,
		regv1.ConditionTypeService,
		regv1.ConditionTypeSecretTLS,
		regv1.ConditionTypeSecretOpaque,
		regv1.ConditionTypeSecretDockerConfigJSON,
		regv1.ConditionTypePvc,
		regv1.ConditionTypeConfigMap,
		regv1.ConditionTypeKeycloakRealm,
	}

	if reg.Spec.Notary.Enabled {
		checkTypes = append(checkTypes, regv1.ConditionTypeNotary)
	}

	if reg.Spec.RegistryService.ServiceType == "Ingress" {
		checkTypes = append(checkTypes, regv1.ConditionTypeIngress)
	}

	return checkTypes
}

func contains(arr []status.ConditionType, ct status.ConditionType) bool {
	for _, a := range arr {
		if a == ct {
			return true
		}
	}
	return false
}
