package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalRegistryCronJob is a scheme of external registry cron job
func ExternalRegistryCronJob(exreg *regv1.ExternalRegistry) *regv1.RegistryCronJob {
	return &regv1.RegistryCronJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      SubresourceName(exreg, SubTypeExternalRegistryCronJob),
			Namespace: exreg.Namespace,
			Labels: map[string]string{
				"app":  "external-registry-cron-job",
				"apps": SubresourceName(exreg, SubTypeExternalRegistryCronJob),
			},
		},
		Spec: regv1.RegistryCronJobSpec{
			JobSpec: regv1.RegistryJobSpec{
				Priority: 0,
				Claim: &regv1.RegistryJobClaim{
					JobType: regv1.JobTypeSynchronizeExtReg,
					HandleObject: corev1.LocalObjectReference{
						Name: exreg.Name,
					},
				},
				TTL: 180,
			},
			Schedule: func(s string) string {
				switch s {
				case "":
					return config.Config.GetString(config.ConfigExternalRegistrySyncPeriod)
				default:
					return s
				}
			}(exreg.Spec.Schedule),
		},
	}
}
