package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalRegistryCronJob is a scheme of external registry cron job
func ExternalRegistryCronJob(exreg *regv1.ExternalRegistry) *regv1.RegistryCronJob {
	labels := make(map[string]string)
	resName := SubresourceName(exreg, SubTypeExternalRegistryCronJob)
	labels["app"] = "external-registry-cron-job"
	labels["apps"] = resName

	schedule := exreg.Spec.Schedule
	if schedule == "" {
		schedule = config.Config.GetString(config.ConfigExternalRegistrySyncPeriod)
	}

	return &regv1.RegistryCronJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: exreg.Namespace,
			Labels:    labels,
		},
		Spec: regv1.RegistryCronJobSpec{
			JobSpec: regv1.RegistryJobSpec{
				Priority: 0,
				SyncRepository: &regv1.RegistryJobSyncRepository{
					ExternalRegistry: corev1.LocalObjectReference{
						Name: exreg.Name,
					},
				},
				TTL: 180,
			},
			Schedule: schedule,
		},
	}
}
