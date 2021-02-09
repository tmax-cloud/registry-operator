package exregctl

import (
	"context"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// UpdateRegistryStatus ...
// If registry status is updated, return true.
func UpdateRegistryStatus(c client.Client, exreg *regv1.ExternalRegistry) (bool, error) {
	reqLogger := logf.Log.WithName("controller_registry").WithValues("Request.Namespace", exreg.Namespace, "Request.Name", exreg.Name)

	if exreg.Status.State == "" {
		if err := initRegistryStatus(c, exreg); err != nil {
			reqLogger.Error(err, "couldn't initialize status")
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func initRegistryStatus(c client.Client, exreg *regv1.ExternalRegistry) error {
	reqLogger := logf.Log.WithName("controller_registry").WithValues("Request.Namespace", exreg.Namespace, "Request.Name", exreg.Name)

	exreg.Status.State = regv1.ExternalRegistryScheduling
	exreg.Status.StateChangedAt = metav1.Now()

	if err := c.Status().Update(context.TODO(), exreg); err != nil {
		reqLogger.Error(err, "couldn't update status")
		return err
	}

	return nil
}
