package replicatectl

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

// UpdateImageReplicateStatus ...
// If image replicate status is updated, return true.
func UpdateImageReplicateStatus(c client.Client, repl *regv1.ImageReplicate) (bool, error) {
	reqLogger := logf.Log.WithName("replicatectl_status").WithValues("Namespace", repl.Namespace, "Name", repl.Name)
	checkTypes := getCheckTypes(repl)

	if len(repl.Status.Conditions) != len(checkTypes) || repl.Status.State == "" {
		if err := initRegistryStatus(c, repl); err != nil {
			return false, err
		}
		return true, nil
	}

	// Check if all subresources are true
	reqLogger.Info("Check if status fields are normal.")
	for _, t := range checkTypes {
		if repl.Status.Conditions.GetCondition(t) == nil {
			reqLogger.Info("Initialize status fields")
			if err := initRegistryStatus(c, repl); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	desiredStatus := regv1.ImageReplicatePending
	switch repl.Status.State {
	case regv1.ImageReplicatePending:
		cond := repl.Status.Conditions.GetCondition(regv1.ConditionTypeImageReplicateRegistryJobProcessing)
		if cond == nil {
			return false, fmt.Errorf("%s condition is not found", regv1.ConditionTypeImageReplicateRegistryJobProcessing)
		}
		if cond.IsTrue() {
			desiredStatus = regv1.ImageReplicateProcessing
		}

	case regv1.ImageReplicateProcessing:
		rjCond := repl.Status.Conditions.GetCondition(regv1.ConditionTypeImageReplicateRegistryJobSuccess)
		if rjCond == nil {
			return false, fmt.Errorf("%s condition is not found", regv1.ConditionTypeImageReplicateRegistryJobSuccess)
		}

		if rjCond.IsUnknown() {
			return false, nil
		}

		if rjCond.IsFalse() {
			desiredStatus = regv1.ImageReplicateFail
			break
		}

		isrCond := repl.Status.Conditions.GetCondition(regv1.ConditionTypeImageReplicateImageSigningSuccess)
		if isrCond == nil && rjCond.IsTrue() {
			desiredStatus = regv1.ImageReplicateSuccess
			break
		}

		if isrCond.IsUnknown() {
			return false, nil
		}

		if isrCond.IsFalse() {
			desiredStatus = regv1.ImageReplicateFail
			break
		}

		if isrCond.IsTrue() {
			desiredStatus = regv1.ImageReplicateSuccess
			break
		}

	default:
		return false, fmt.Errorf("invalid state: %s", repl.Status.State)
	}

	reqLogger.Info("desiredStatus", "status", desiredStatus)

	// Chcck if current status is desired status. If does not same, update the status.
	reqLogger.Info("Check if current status is desired status.")
	if repl.Status.State == desiredStatus {
		return false, nil
	}

	reqLogger.Info(fmt.Sprintf("Current Status(%s) -> Desired Status(%s)", string(repl.Status.State), string(desiredStatus)))

	target := repl.DeepCopy()
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

func initRegistryStatus(c client.Client, repl *regv1.ImageReplicate) error {
	reqLogger := logf.Log.WithName("replicatectl_status").WithValues("Namespace", repl.Namespace, "Name", repl.Name)

	if repl.Status.Conditions == nil {
		repl.Status.Conditions = status.NewConditions()
	}

	// Set Conditions
	checkTypes := getCheckTypes(repl)
	for _, t := range checkTypes {
		if repl.Status.Conditions.GetCondition(t) == nil {
			reqLogger.Info("New Condition: " + string(t))
			newCondition := status.Condition{Type: t, Status: corev1.ConditionUnknown}
			repl.Status.Conditions.SetCondition(newCondition)
		}
	}

	for _, t := range repl.Status.Conditions {
		if !contains(checkTypes, t.Type) {
			reqLogger.Info("Removed Condition: " + string(t.Type))
			repl.Status.Conditions.RemoveCondition(t.Type)
		}
	}

	if repl.Status.State == "" {
		repl.Status.State = regv1.ImageReplicatePending
		repl.Status.StateChangedAt = metav1.Now()
	}

	if err := c.Status().Update(context.TODO(), repl); err != nil {
		reqLogger.Error(err, "couldn't update status")
		return err
	}

	return nil
}

func getCheckTypes(repl *regv1.ImageReplicate) []status.ConditionType {
	checkTypes := []status.ConditionType{
		regv1.ConditionTypeImageReplicateRegistryJobExist,
		regv1.ConditionTypeImageReplicateRegistryJobProcessing,
		regv1.ConditionTypeImageReplicateRegistryJobSuccess,
	}

	if repl.Spec.ToImage.RegistryType != regv1.RegistryTypeHpcdRegistry {
		checkTypes = append(checkTypes, regv1.ConditionTypeImageReplicateSynchronized)
	}

	if repl.Spec.Signer != "" {
		checkTypes = append(checkTypes,
			regv1.ConditionTypeImageReplicateImageSignRequestExist,
			regv1.ConditionTypeImageReplicateImageSigning,
			regv1.ConditionTypeImageReplicateImageSigningSuccess,
		)
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
