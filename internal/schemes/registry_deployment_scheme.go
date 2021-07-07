package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"path"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// RegistryPVCMountPath is registry's default mount path to pvc
	RegistryPVCMountPath = "/var/lib/registry"
	// RegistryEnvKeyStorageMaintenance is registry storage maintenance config
	RegistryEnvKeyStorageMaintenance = "REGISTRY_STORAGE_MAINTENANCE"
	// RegistryEnvValueStorageMaintenance sets readonly
	RegistryEnvValueStorageMaintenance = `{"readonly":{"enabled":true}}`

	configMapMountPath = "/etc/docker/registry"

	registryTLSCrtPath = "/certs/registry/tls.crt"
	registryTLSKeyPath = "/certs/registry/tls.key"
	authTokenKeyPath   = "/certs/rootca/ca.crt"
)

// Deployment is a scheme of registry deployment
func Deployment(reg *regv1.Registry, auth *regv1.AuthConfig) (*appsv1.Deployment, error) {
	var resName, pvcMountPath, pvcName, configMapName string
	resName = SubresourceName(reg, SubTypeRegistryDeployment)
	label, labelSelector := map[string]string{}, map[string]string{}
	label["app"] = "registry"
	label["apps"] = resName
	label[resName] = "lb"

	for k, v := range label {
		labelSelector[k] = v
	}

	// Set label
	for k, v := range reg.Spec.RegistryDeployment.Labels {
		label[k] = v
	}

	// Set labelSelector
	for k, v := range reg.Spec.RegistryDeployment.Selector.MatchLabels {
		labelSelector[k] = v
	}

	// Set mountpath
	if len(reg.Spec.PersistentVolumeClaim.MountPath) == 0 {
		pvcMountPath = RegistryPVCMountPath
	} else {
		pvcMountPath = reg.Spec.PersistentVolumeClaim.MountPath
	}

	// Set pvc
	if reg.Spec.PersistentVolumeClaim.Exist != nil {
		pvcName = reg.Spec.PersistentVolumeClaim.Exist.PvcName
	} else {
		pvcName = SubresourceName(reg, SubTypeRegistryPVC)
	}

	// Set config yaml
	if len(reg.Spec.CustomConfigYml) != 0 {
		configMapName = reg.Spec.CustomConfigYml
	} else {
		configMapName = SubresourceName(reg, SubTypeRegistryConfigmap)
	}

	registryImage := reg.Spec.Image
	if registryImage == "" {
		registryImage = config.Config.GetString(config.ConfigRegistryImage)
	}

	cpuRequest := *reg.Spec.RegistryDeployment.Resources.Requests.Cpu()
	memoryRequest := *reg.Spec.RegistryDeployment.Resources.Requests.Memory()
	cpuLimit := *reg.Spec.RegistryDeployment.Resources.Limits.Cpu()
	memoryLimit := *reg.Spec.RegistryDeployment.Resources.Limits.Memory()
	if cpuRequest.IsZero() {
		cpuRequest = resource.MustParse(config.Config.GetString(config.ConfigRegistryCPU))
	}
	if memoryRequest.IsZero() {
		memoryRequest = resource.MustParse(config.Config.GetString(config.ConfigRegistryMemory))
	}
	if cpuLimit.IsZero() {
		cpuLimit = resource.MustParse(config.Config.GetString(config.ConfigRegistryCPU))
	}
	if memoryLimit.IsZero() {
		memoryLimit = resource.MustParse(config.Config.GetString(config.ConfigRegistryMemory))
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resName,
			Namespace: reg.Namespace,
			Labels:    label,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels:      labelSelector,
				MatchExpressions: reg.Spec.RegistryDeployment.Selector.MatchExpressions,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},
				Spec: corev1.PodSpec{
					Tolerations:  reg.Spec.RegistryDeployment.Tolerations,
					NodeSelector: reg.Spec.RegistryDeployment.NodeSelector,
					Containers: []corev1.Container{
						{
							Image: registryImage,
							Name:  "registry",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    cpuLimit,
									corev1.ResourceMemory: memoryLimit,
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    cpuRequest,
									corev1.ResourceMemory: memoryRequest,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "tls",
									ContainerPort: 443,
									Protocol:      "TCP",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "REGISTRY_AUTH",
									Value: "token",
								},
								{
									Name:  "REGISTRY_AUTH_TOKEN_REALM",
									Value: auth.Realm,
								},
								{
									Name:  "REGISTRY_AUTH_TOKEN_SERVICE",
									Value: auth.Service,
								},
								{
									Name:  "REGISTRY_AUTH_TOKEN_ISSUER",
									Value: auth.Issuer,
								},
								{
									Name:  "REGISTRY_AUTH_TOKEN_ROOTCERTBUNDLE",
									Value: authTokenKeyPath,
								},
								{
									Name:  "REGISTRY_HTTP_ADDR",
									Value: "0.0.0.0:443",
								},
								{
									Name:  "REGISTRY_HTTP_TLS_CERTIFICATE",
									Value: registryTLSCrtPath,
								},
								{
									Name:  "REGISTRY_HTTP_TLS_KEY",
									Value: registryTLSKeyPath,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "registry",
									MountPath: pvcMountPath,
								},
								{
									Name:      "config",
									MountPath: configMapMountPath,
									ReadOnly:  true,
								},
								{
									Name:      "https",
									MountPath: path.Dir(registryTLSKeyPath),
									ReadOnly:  true,
								},
								{
									Name:      "auth",
									MountPath: path.Dir(authTokenKeyPath),
									ReadOnly:  true,
								},
							},
							ReadinessProbe: &corev1.Probe{
								PeriodSeconds:       10,
								TimeoutSeconds:      1,
								InitialDelaySeconds: 5,
								SuccessThreshold:    1,
								FailureThreshold:    3,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/",
										Port:   intstr.IntOrString{IntVal: 443},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								PeriodSeconds:       10,
								TimeoutSeconds:      1,
								InitialDelaySeconds: 200,
								SuccessThreshold:    1,
								FailureThreshold:    3,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/",
										Port:   intstr.IntOrString{IntVal: 443},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "registry",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
								},
							},
						},
						{
							Name: "https",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: SubresourceName(reg, SubTypeRegistryTLSSecret),
								},
							},
						},
						{
							Name: "auth",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "registry-token-key",
								},
							},
						},
					},
				},
			},
		},
	}

	if reg.Spec.ReadOnly {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  RegistryEnvKeyStorageMaintenance,
				Value: RegistryEnvValueStorageMaintenance,
			},
		)
	}

	if config.Config.GetString(config.ConfigRegistryImagePullSecret) != "" {
		deployment.Spec.Template.Spec.ImagePullSecrets = append(deployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: config.Config.GetString(config.ConfigRegistryImagePullSecret)})
	}

	return deployment, nil
}
