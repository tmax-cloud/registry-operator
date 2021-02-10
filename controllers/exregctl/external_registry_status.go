package exregctl

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// UpdateRegistryStatus ...
// If registry status is updated, return true.
func UpdateRegistryStatus(c client.Client, exreg *regv1.ExternalRegistry) (bool, error) {
	reqLogger := logf.Log.WithName("exregctl_status").WithValues("Request.Namespace", exreg.Namespace, "Request.Name", exreg.Name)
	falseTypes := []status.ConditionType{}
	checkTypes := getCheckTypes(exreg)

	if len(exreg.Status.Conditions) != len(checkTypes) || exreg.Status.State == "" {
		if err := initRegistryStatus(c, exreg); err != nil {
			return false, err
		}
		return true, nil
	}

	// Check if all subresources are true
	reqLogger.Info("Check if status fields are normal.")
	for _, t := range checkTypes {
		if exreg.Status.Conditions.IsUnknownFor(t) {
			reqLogger.Info("Initialize status fields")
			if err := initRegistryStatus(c, exreg); err != nil {
				return false, err
			}
			return true, nil
		} else if exreg.Status.Conditions.IsFalseFor(t) {
			falseTypes = append(falseTypes, t)
		}
	}

	for _, t := range falseTypes {
		reqLogger.Info("false_type", "false", t)
	}

	desiredStatus := regv1.ExternalRegistryReady

	if len(falseTypes) > 0 {
		desiredStatus = regv1.ExternalRegistryNotReady
	}

	reqLogger.Info("desiredStatus", "status", desiredStatus)

	// Chcck if current status is desired status. If does not same, update the status.
	reqLogger.Info("Check if current status is desired status.")
	if exreg.Status.State == desiredStatus {
		return false, nil
	}

	reqLogger.Info(fmt.Sprintf("Current Status(%s) -> Desired Status(%s)", string(exreg.Status.State), string(desiredStatus)))

	target := exreg.DeepCopy()
	target.Status.State = desiredStatus
	target.Status.StateChangedAt = metav1.Now()

	// Patch the status to desired status.
	reqLogger.Info("Status update.")
	if err := c.Status().Update(context.TODO(), target); err != nil {
		reqLogger.Error(err, "failed to update status")
		return false, err
	}

	return true, nil
}

func initRegistryStatus(c client.Client, exreg *regv1.ExternalRegistry) error {
	reqLogger := logf.Log.WithName("exregctl_status").WithValues("Request.Namespace", exreg.Namespace, "Request.Name", exreg.Name)

	if exreg.Status.Conditions == nil {
		exreg.Status.Conditions = status.NewConditions()
	}

	// Set Conditions
	checkTypes := getCheckTypes(exreg)
	for _, t := range checkTypes {
		if exreg.Status.Conditions.GetCondition(t) == nil {
			reqLogger.Info("New Condition: " + string(t))
			newCondition := status.Condition{Type: t, Status: corev1.ConditionFalse}
			exreg.Status.Conditions.SetCondition(newCondition)
		}
	}

	for _, t := range exreg.Status.Conditions {
		if !contains(checkTypes, t.Type) {
			exreg.Status.Conditions.RemoveCondition(t.Type)
		}
	}

	exreg.Status.State = regv1.ExternalRegistryPending
	exreg.Status.StateChangedAt = metav1.Now()

	if err := c.Status().Update(context.TODO(), exreg); err != nil {
		reqLogger.Error(err, "couldn't update status")
		return err
	}

	return nil
}

func getCheckTypes(exreg *regv1.ExternalRegistry) []status.ConditionType {
	checkTypes := []status.ConditionType{
		regv1.ConditionTypeExRegistryCronJobExist,
		regv1.ConditionTypeExRegistryInitialized,
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
