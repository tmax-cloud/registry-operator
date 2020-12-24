package schemes

import (
	"path"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	serverImage      = "tmaxcloudck/notary_server:0.6.2-rc1"
	serverTLSCrtPath = "/certs/server/tls.crt"
	serverTLSKeyPath = "/certs/server/tls.key"
	serverRootCAPath = "/certs/rootca/ca.crt"
)

func NotaryServerPod(notary *regv1.Notary) *corev1.Pod {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotaryServerPod)
	labels["app"] = "notary-server"
	labels["apps"] = resName
	labels[resName] = "lb"

	mode := int32(511)

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: notary.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "notary-server",
					Image:           serverImage,
					ImagePullPolicy: corev1.PullAlways,
					Command:         []string{"/usr/bin/env", "sh"},
					Args:            []string{"-c", "/var/lib/notary/migrations/migrate.sh && notary-server -config=/var/lib/notary/fixtures/custom/server-config.json"},
					Env: []corev1.EnvVar{
						{
							Name:  "NOTARY_SERVER_SERVER_HTTP_ADDR",
							Value: ":4443",
						},
						{
							Name:  "NOTARY_SERVER_SERVER_TLS_CERT_FILE",
							Value: serverTLSCrtPath,
						},
						{
							Name:  "NOTARY_SERVER_SERVER_TLS_KEY_FILE",
							Value: serverTLSKeyPath,
						},
						// {
						// 	Name:  "NOTARY_SERVER_SERVER_CLIENT_CA_FILE",
						// 	Value: serverRootCAPath,
						// },
						{
							Name:  "NOTARY_SERVER_LOGGING_LEVEL",
							Value: "debug",
						},
						{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_TYPE",
							Value: "remote",
						},
						{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_HOSTNAME",
							Value: utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotarySignerService), notary.Namespace),
						},
						{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_PORT",
							Value: "7899",
						},
						{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_TLS_CA_FILE",
							Value: serverRootCAPath,
						},
						{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_KEY_ALGORITHM",
							Value: "ecdsa",
						},
						{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_TLS_CLIENT_CERT",
							Value: serverTLSCrtPath,
						},
						{
							Name:  "NOTARY_SERVER_TRUST_SERVICE_TLS_CLIENT_KEY",
							Value: serverTLSKeyPath,
						},
						{
							Name:  "NOTARY_SERVER_AUTH_TYPE",
							Value: "token",
						},
						{
							Name:  "NOTARY_SERVER_AUTH_OPTIONS_REALM",
							Value: notary.Spec.AuthConfig.Realm,
						},
						{
							Name:  "NOTARY_SERVER_AUTH_OPTIONS_SERVICE",
							Value: notary.Spec.AuthConfig.Service,
						},
						{
							Name:  "NOTARY_SERVER_AUTH_OPTIONS_ISSUER",
							Value: notary.Spec.AuthConfig.Issuer,
						},
						{
							Name:  "NOTARY_SERVER_AUTH_OPTIONS_ROOTCERTBUNDLE",
							Value: serverRootCAPath,
						},
						{
							Name:  "NOTARY_SERVER_STORAGE_BACKEND",
							Value: "mysql",
						},
						{
							Name:  "NOTARY_SERVER_STORAGE_DB_URL",
							Value: "server@tcp(" + utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotaryDBService), notary.Namespace) + ":3306)/notaryserver?parseTime=True",
						},
						{
							Name:  "MIGRATIONS_PATH",
							Value: "/var/lib/notary/migrations/server/mysql",
						},
						{
							Name:  "DB_URL",
							Value: "mysql://server@tcp(" + utils.BuildServiceHostname(SubresourceName(notary, SubTypeNotaryDBService), notary.Namespace) + ":3306)/notaryserver",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "server-tls",
							MountPath: path.Dir(serverTLSCrtPath),
						},
						{
							Name:      "root-ca",
							MountPath: path.Dir(serverRootCAPath),
						},
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 4443,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "server-tls",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: &mode,
							SecretName:  SubresourceName(notary, SubTypeNotaryServerSecret),
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

	return pod
}
