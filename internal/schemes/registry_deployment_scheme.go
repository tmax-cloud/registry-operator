package schemes

import (
	"path"
	"strconv"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	"github.com/tmax-cloud/registry-operator/internal/common/config"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// RegistryPVCMountPath is registry's default mount path to pvc
	RegistryPVCMountPath = "/var/lib/registry"

	// DefaultResourceCPU is default resource cpu requirement
	DefaultResourceCPU = "0.1"
	// DefaultResourceMemory is default resource memory requirement
	DefaultResourceMemory = "512Mi"
	configMapMountPath    = "/etc/docker/registry"

	registryTLSCrtPath = "/certs/registry/tls.crt"
	registryTLSKeyPath = "/certs/registry/tls.key"
	registryRootCAPath = "/certs/rootca/ca.crt"
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
		pvcName = regv1.K8sPrefix + reg.Name
	}

	// Set config yaml
	if len(reg.Spec.CustomConfigYml) != 0 {
		configMapName = reg.Spec.CustomConfigYml
	} else {
		configMapName = regv1.K8sPrefix + reg.Name
	}

	if _, err := certs.GetRootCert(reg.Namespace); err != nil {
		return nil, err
	}

	registryImage := reg.Spec.Image
	if registryImage == "" {
		registryImage = config.Config.GetString(config.ConfigRegistryImage)
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
									corev1.ResourceCPU:    resource.MustParse(DefaultResourceCPU),
									corev1.ResourceMemory: resource.MustParse(DefaultResourceMemory),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(DefaultResourceCPU),
									corev1.ResourceMemory: resource.MustParse(DefaultResourceMemory),
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          RegistryPortName,
									ContainerPort: RegistryTargetPort,
									Protocol:      RegistryPortProtocol,
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
									Value: registryRootCAPath,
								},
								{
									Name:  "REGISTRY_HTTP_ADDR",
									Value: string("0.0.0.0:") + strconv.Itoa(RegistryTargetPort),
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
								},
								{
									Name:      "tls",
									MountPath: path.Dir(registryTLSKeyPath),
								},
								{
									Name:      "rootca",
									MountPath: path.Dir(registryRootCAPath),
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
										Port:   intstr.IntOrString{IntVal: RegistryTargetPort},
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
										Port:   intstr.IntOrString{IntVal: RegistryTargetPort},
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
							Name: "tls",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: regv1.K8sPrefix + regv1.TLSPrefix + reg.Name,
								},
							},
						},
						{
							Name: "rootca",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: certs.RootCASecretName,
								},
							},
						},
					},
				},
			},
		},
	}

	if !reg.Spec.RegistryDeployment.Resources.Limits.Cpu().IsZero() {
		deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = *reg.Spec.RegistryDeployment.Resources.Limits.Cpu()
	}
	if !reg.Spec.RegistryDeployment.Resources.Limits.Memory().IsZero() {
		deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = *reg.Spec.RegistryDeployment.Resources.Limits.Memory()
	}
	if !reg.Spec.RegistryDeployment.Resources.Requests.Cpu().IsZero() {
		deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = *reg.Spec.RegistryDeployment.Resources.Requests.Cpu()
	}
	if !reg.Spec.RegistryDeployment.Resources.Requests.Memory().IsZero() {
		deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = *reg.Spec.RegistryDeployment.Resources.Requests.Memory()
	}

	if config.Config.GetString(config.ConfigRegistryImagePullSecret) != "" {
		deployment.Spec.Template.Spec.ImagePullSecrets = append(deployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: config.Config.GetString(config.ConfigRegistryImagePullSecret)})
	}

	return deployment, nil
}
