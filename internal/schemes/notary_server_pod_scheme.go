package schemes

import (
	"path"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ServerImage      = "tmaxcloudck/notary_server:0.6.2-rc1"
	ServerTlsCrtPath = "/certs/server/tls.crt"
	ServerTlsKeyPath = "/certs/server/tls.key"
	ServerRootCAPath = "/certs/rootca/root-ca.crt"
)

func NotaryServerPod(notary *regv1.Notary) *corev1.Pod {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotaryServerPod)
	labels["app"] = "notary-server"
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
					Image:   ServerImage,
					Command: []string{"/usr/bin/env", "sh"},
					Args:    []string{"-c", "/notary/server/migrate.sh && notary-server -config=/notary/server/server-config.json"},
					Env: []corev1.EnvVar{
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_TYPE",
							Value: "remote",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_HOSTNAME",
							Value: utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotarySignerService), notary.Namespace),
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_PORT",
							Value: "7899",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_TLS_CA_FILE",
							Value: ServerRootCAPath,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_KEY_ALGORITHM",
							Value: "ecdsa",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_TLS_CLIENT_CERT",
							Value: ServerTlsCrtPath,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_TLS_CLIENT_KEY",
							Value: ServerTlsKeyPath,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_AUTH_TYPE",
							Value: "token",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_AUTH_OPTIONS_REALM",
							Value: notary.Spec.AuthConfig.Realm,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_AUTH_OPTIONS_SERVICE",
							Value: notary.Spec.AuthConfig.Service,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_AUTH_OPTIONS_ISSUER",
							Value: notary.Spec.AuthConfig.Issuer,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_AUTH_OPTIONS_ROOTCERTBUNDLE",
							Value: ServerRootCAPath,
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_STORAGE_BACKEND",
							Value: "mysql",
						},
						corev1.EnvVar{
							Name:  "NOTARY_SERVER_STORAGE_DB_URL",
							Value: "server@tcp(" + utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotaryDBService), notary.Namespace) + ":3306)/notaryserver?parseTime=True",
						},
						corev1.EnvVar{
							Name:  "MIGRATIONS_PATH",
							Value: "/var/lib/notary/migrations/server/mysql",
						},
						corev1.EnvVar{
							Name:  "DB_URL",
							Value: "mysql://server@tcp(" + utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotaryDBService), notary.Namespace) + ":3306)/notaryserver",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      "server-tls",
							MountPath: path.Dir(ServerTlsCrtPath),
						},
						corev1.VolumeMount{
							Name:      "root-ca",
							MountPath: path.Dir(ServerRootCAPath),
						},
					},
					Ports: []corev1.ContainerPort{
						corev1.ContainerPort{
							ContainerPort: 4443,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				corev1.Volume{
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: SubresourceName(notary, SubTypeNotaryServerSecret),
						},
					},
				},
				corev1.Volume{
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
