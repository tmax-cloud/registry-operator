package notaryctl

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"
	"sigs.k8s.io/controller-runtime/pkg/client"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func UpdateNotaryStatus(c client.Client, not *regv1.Notary) (bool, error) {
	reqLogger := logf.Log.WithName("controller_notary").WithValues("Request.Namespace", not.Namespace, "Request.Name", not.Name)
	falseTypes := []status.ConditionType{}
	checkTypes := getCheckTypes(not)

	if len(not.Status.Conditions) != len(checkTypes) {
		if err := initNotaryStatus(c, not); err != nil {
			return false, err
		}
		return true, nil
	}

	// Check if all subresources are true
	reqLogger.Info("Check if status fields are normal.")
	for _, t := range checkTypes {
		if not.Status.Conditions.IsUnknownFor(t) {
			reqLogger.Info("Initialize status fields")
			if err := initNotaryStatus(c, not); err != nil {
				return false, err
			}
			return true, nil
		} else if not.Status.Conditions.IsFalseFor(t) {
			falseTypes = append(falseTypes, t)
		}
	}

	for _, t := range falseTypes {
		reqLogger.Info("false_type", "false", t)
	}
	return false, nil
}

func initNotaryStatus(c client.Client, not *regv1.Notary) error {
	reqLogger := logf.Log.WithName("controller_registry").WithValues("Request.Namespace", not.Namespace, "Request.Name", not.Name)

	if not.Status.Conditions == nil {
		not.Status.Conditions = status.NewConditions()
	}

	// Set Conditions
	checkTypes := getCheckTypes(not)
	for _, t := range checkTypes {
		reqLogger.Info("Check Type: " + string(t))
		if not.Status.Conditions.GetCondition(t) == nil {
			newCondition := status.Condition{Type: t, Status: corev1.ConditionFalse}
			not.Status.Conditions.SetCondition(newCondition)
		}
	}

	for _, t := range not.Status.Conditions {
		if !contains(checkTypes, t.Type) {
			not.Status.Conditions.RemoveCondition(t.Type)
		}
	}

	err := c.Status().Update(context.TODO(), not)
	if err != nil {
		reqLogger.Error(err, "cannot update status")
		return err
	}

	return nil
}

func getCheckTypes(not *regv1.Notary) []status.ConditionType {
	checkTypes := []status.ConditionType{
		regv1.ConditionTypeNotaryDBPod,
		regv1.ConditionTypeNotaryDBPVC,
		regv1.ConditionTypeNotaryDBService,
		regv1.ConditionTypeNotaryServerPod,
		regv1.ConditionTypeNotaryServerSecret,
		regv1.ConditionTypeNotaryServerService,
		regv1.ConditionTypeNotarySignerPod,
		regv1.ConditionTypeNotarySignerSecret,
		regv1.ConditionTypeNotarySignerService,
	}

	if not.Spec.ServiceType == "Ingress" {
		checkTypes = append(checkTypes, regv1.ConditionTypeNotaryServerIngress)
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
