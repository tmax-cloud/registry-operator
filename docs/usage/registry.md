# `Registry` Usage

## What is it?

`Registry` is a resource that lets you create and manage container image repositories.

## How to create

### spec fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.image`                                 | No  | string            | Registry's image name |
|`spec.description`                           | No  | string            | Description for registry |
|`spec.loginId`                               | Yes | string            | Login ID for registry |
|`spec.loginPassword`                         | Yes | string            | Login password for registry |
|`spec.readOnly`                              | No  | bool              | If ReadOnly is true, clients will not be allowed to write(push) to the registry. |
|`spec.notary`                                | No  | object            | Settings for notary service |
|`spec.customConfigYml`                       | No  | string            | The name of the configmap where the registry config.yml content |
|`spec.registryDeployment`                    | No  | object            | Settings for registry's deployemnt |
|`spec.service`                               | Yes | object            | Service type to expose registry |
|`spec.persistentVolumeClaim`                 | Yes | object            | Settings for registry pvc |

### spec.notary fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.notary.enabled`                        | Yes | bool              | Activate notary service to sign images |
|`spec.notary.serviceType`                    | Yes | string            | Use Ingress or LoadBalancer |
|`spec.notary.persistentVolumeClaim`          | Yes | object            | Settings for notary pvc |

### spec.notary.persistentVolumeClaim fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.notary.persistentVolumeClaim.exist`    | No  | object            | Use exist pvc |
|`spec.notary.persistentVolumeClaim.create`   | No  | object            | Create new pvc |

### spec.notary.persistentVolumeClaim.exist and spec.notary.persistentVolumeClaim.create fields

|Key|Required|Type|Description|
|:----------------------------------------------------------:|-----|-------------------|-----|
|`spec.notary.persistentVolumeClaim.exist.pvcName`           | No  | string            | PVC's name you have created |
|`spec.notary.persistentVolumeClaim.create.accessModes`      | Yes | array             | Each PV's access modes are set to the specific modes supported by that particular volume. |
|`spec.notary.persistentVolumeClaim.create.storageSize`      | Yes | string            | Desired storage size like "10Gi" |
|`spec.notary.persistentVolumeClaim.create.storageClassName` | Yes | string            | StorageClassName like "csi-cephfs-sc" |
|`spec.notary.persistentVolumeClaim.create.deleteWithPvc`    | No  | bool              | Delete the pvc as well when this registry is deleted (default: false) |

### spec.registryDeployment fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|----------------------|-----|
|`spec.registryDeployment.labels`             | No  | map[string]string    | Deployment's label |
|`spec.registryDeployment.nodeSelector`       | No  | map[string]string    | Registry pod's node selector |
|`spec.registryDeployment.selector`           | No  | [metav1.LabelSelector](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#LabelSelector) | Deployment's label selector |
|`spec.registryDeployment.tolerations`        | No  | [][corev1.Toleration](https://pkg.go.dev/k8s.io/api/core/v1?utm_source=godoc#Toleration) | Deployment's toleration configuration |
|`spec.registryDeployment.resources`          | No  | [corev1.ResourceRequirements](https://pkg.go.dev/k8s.io/api/core/v1#ResourceRequirements) | Deployment's resource requirements |

### spec.service fields

|Key|Required|Type|Description|
|:----------------------------------------------------------:|-----|-------------------|-----|
|`spec.service.serviceType`                                  | Yes | string            | Use Ingress or LoadBalancer |

### spec.persistentVolumeClaim fields

|Key|Required|Type|Description|
|:----------------------------------------------------------:|-----|-------------------|-----|
|`spec.persistentVolumeClaim.mountPath`                      | No  | string            | Registry's pvc mount path (default: /var/lib/registry) |
|`spec.persistentVolumeClaim.exist`                          | No  | string            |  |
|`spec.persistentVolumeClaim.create`                         | No  | string            |  |

### spec.persistentVolumeClaim.exist and spec.persistentVolumeClaim.create fields

|Key|Required|Type|Description|
|:----------------------------------------------------------:|-----|-------------------|-----|
|`spec.persistentVolumeClaim.exist.pvcName`                  | No  | string            | PVC's name you have created |
|`spec.persistentVolumeClaim.create.accessModes`             | Yes | array             | Each PV's access modes are set to the specific modes supported by that particular volume. |
|`spec.persistentVolumeClaim.create.storageSize`             | Yes | string            | Desired storage size like "10Gi" |
|`spec.persistentVolumeClaim.create.storageClassName`        | Yes | string            | StorageClassName like "csi-cephfs-sc" |
|`spec.persistentVolumeClaim.create.deleteWithPvc`           | No  | bool              | Delete the pvc as well when this registry is deleted (default: false) |

## Example

Reference: [Test Example](../../config/samples/tmax.io_v1_registry.yaml)

## Result

* State(status.phase)
  * Creating: registry's subresources is being created.
  * NotReady: container is not ready.
  * Running: All subresources are created and registry container is running.

* State(status.phase) Transition
  1) (None) -> Creating: Creating subresources is started.
  2) Creating -> NotReady: All subresources are created but the container is not ready.
  3) NotReady -> Running: Registry container is running
  4) Running -> Creating: The spec of a subresource has changed or has been deleted.

* Created Subresource Names in the namespace
  * Service: hpcd-{REGISTRY_NAME}
  * PVC: hpcd-{REGISTRY_NAME}
  * Deployment: hpcd-{REGISTRY_NAME}
  * Secret(Registry Info, type: Opaque): hpcd-{REGISTRY_NAME}
  * Secret(ImagePullSecret, type: kubernetes.io/dockerconfigjson): hpcd-registry-{REGISTRY_NAME}
  * Secret(tls, type: kubernetes.io/tls): hpcd-tls-{REGISTRY_NAME}

  * If `spec.customConfigYml` is not set
    * CM: hpcd-{REGISTRY_NAME}
  * If `spec.service.serviceType` is Ingress
    * Ingress: hpcd-{REGISTRY_NAME}
  * If `spec.notary.enabled` is true
    * Notary: {REGISTRY_NAME}
