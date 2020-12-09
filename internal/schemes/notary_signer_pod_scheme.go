package schemes

import (
	"path"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SignerImage      = "tmaxcloudck/notary_signer:0.6.2-rc1"
	SignerTlsCrtPath = "/certs/signer/tls.crt"
	SignerTlsKeyPath = "/certs/signer/tls.key"
	SignerRootCAPath = "/certs/rootca/root-ca.crt"
)

func NotarySignerPod(notary *regv1.Notary) *corev1.Pod {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotarySignerPod)
	labels["app"] = "notary-signer"
	labels["apps"] = resName
	labels[resName] = "lb"

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: notary.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:    "notary-server",
					Image:   SignerImage,
					Command: []string{"/usr/bin/env", "sh"},
					Args:    []string{"-c", "/notary/signer/migrate.sh && notary-signer -config=/notary/signer/signer-config.json"},
					Env: []corev1.EnvVar{
						corev1.EnvVar{
							Name:  "NOTARY_SIGNER_STORAGE_BACKEND",
							Value: "mysql",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SIGNER_STORAGE_DB_URL",
							Value: "signer@tcp(" + utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotaryDBService), notary.Namespace) + ":3306)/notarysigner?parseTime=True",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SIGNER_SERVER_HTTP_ADDR",
							Value: ":4444",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SIGNER_SERVER_GRPC_ADDR",
							Value: ":7899",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SIGNER_SERVER_TLS_CERT_FILE",
							Value: SignerTlsCrtPath,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SIGNER_SERVER_TLS_KEY_FILE",
							Value: SignerTlsKeyPath,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SIGNER_SERVER_CLIENT_CA_FILE",
							Value: SignerRootCAPath,
						},
						corev1.EnvVar{
							Name:  "MIGRATIONS_PATH",
							Value: "/var/lib/notary/migrations/server/mysql",
						},
						corev1.EnvVar{
							Name:  "DB_URL",
							Value: "mysql://signer@tcp(" + utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotaryDBService), notary.Namespace) + ":3306)/notarysigner",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      "signer-tls",
							MountPath: path.Dir(SignerTlsCrtPath),
						},
						corev1.VolumeMount{
							Name:      "root-ca",
							MountPath: path.Dir(SignerRootCAPath),
						},
					},
					Ports: []corev1.ContainerPort{
						corev1.ContainerPort{
							ContainerPort: 4444,
						},
						corev1.ContainerPort{
							ContainerPort: 7899,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "signer-tls",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: SubresourceName(notary, SubTypeNotarySignerSecret),
						},
					},
				},
				corev1.Volume{
					Name: "root-ca",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: notary.Spec.RootCASecret,
						},
					},
				},
			},
		},
	}

	return pod
}
