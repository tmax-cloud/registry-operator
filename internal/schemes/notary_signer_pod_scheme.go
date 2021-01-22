package schemes

import (
	"path"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	signerTLSCrtPath = "/certs/signer/tls.crt"
	signerTLSKeyPath = "/certs/signer/tls.key"
	signerRootCAPath = "/certs/rootca/ca.crt"
)

func NotarySignerPod(notary *regv1.Notary) *corev1.Pod {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotarySignerPod)
	labels["app"] = "notary-signer"
	labels["apps"] = resName
	labels[resName] = "lb"

	mode := int32(511)

	signerImage := config.Config.GetString("notary.signer.image")

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: notary.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "notary-signer",
					Image:           signerImage,
					ImagePullPolicy: corev1.PullAlways,
					Command:         []string{"/usr/bin/env", "sh"},
					Args:            []string{"-c", "/var/lib/notary/migrations/migrate.sh && notary-signer -config=/var/lib/notary/fixtures/custom/signer-config.json"},
					Env: []corev1.EnvVar{
						{
							Name:  "NOTARY_SIGNER_LOGGING_LEVEL",
							Value: "debug",
						},
						{
							Name:  "NOTARY_SIGNER_STORAGE_BACKEND",
							Value: "mysql",
						},
						{
							Name:  "NOTARY_SIGNER_STORAGE_DB_URL",
							Value: "signer@tcp(" + utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotaryDBService), notary.Namespace) + ":3306)/notarysigner?parseTime=True",
						},
						{
							Name:  "NOTARY_SIGNER_SERVER_HTTP_ADDR",
							Value: ":4444",
						},
						{
							Name:  "NOTARY_SIGNER_SERVER_GRPC_ADDR",
							Value: ":7899",
						},
						{
							Name:  "NOTARY_SIGNER_SERVER_TLS_CERT_FILE",
							Value: signerTLSCrtPath,
						},
						{
							Name:  "NOTARY_SIGNER_SERVER_TLS_KEY_FILE",
							Value: signerTLSKeyPath,
						},
						{
							Name:  "NOTARY_SIGNER_SERVER_CLIENT_CA_FILE",
							Value: signerRootCAPath,
						},
						{
							Name:  "MIGRATIONS_PATH",
							Value: "/var/lib/notary/migrations/signer/mysql",
						},
						{
							Name:  "DB_URL",
							Value: "mysql://signer@tcp(" + utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotaryDBService), notary.Namespace) + ":3306)/notarysigner",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "signer-tls",
							MountPath: path.Dir(signerTLSCrtPath),
						},
						{
							Name:      "root-ca",
							MountPath: path.Dir(signerRootCAPath),
						},
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 4444,
						},
						{
							ContainerPort: 7899,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "signer-tls",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: &mode,
							SecretName:  SubresourceName(notary, SubTypeNotarySignerSecret),
						},
					},
				},
				{
					Name: "root-ca",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: &mode,
							SecretName:  notary.Spec.RootCASecret,
						},
					},
				},
			},
		},
	}

	if config.Config.GetString("notary.signer.image_pull_secret") != "" {
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: config.Config.GetString("notary.signer.image_pull_secret")})
	}

	return pod
}
