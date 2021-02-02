# `ImageSignRequest` Usage

## What is it?

`ImageSignRequest` is a request to sign image by specified signer. Registry that notary service is enabled can request sign images.

## How to create

### spec fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.image`                                 | Yes | string            | Image name to sign (example: docker.io/library/alpine:3) |
|`spec.signer`                                | Yes | string            | ImageSigner's metadata name to sign image |
|`spec.registryLogin`                         | No  | object            | Secrets to login registry|

### spec.registryLogin fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.registryLogin.dcjSecretName`           | Yes | string            | Registry's imagePullSecret for login. If you don't have dockerconfigjson type's secret in this namespace, you should refer to <https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/> to make it first. |
|`spec.registryLogin.certSecretName`          | No  | string            | If you want to trust registry's certificate, enter certifiacete's secret name |

## Example

Reference: [Test Example](../../config/samples/tmax.io_v1_imagesignrequest.yaml)

## Result

* State(status.imageSignResponse.result)
  * Signing: registry's subresources is being created.
  * Success: container is not ready.
  * Fail: All subresources are created and registry container is running.

* State(status.imageSignResponse.result) Transition
  1) (None) -> Signing: Signing an image is started.
  2) Signing -> Success: Signing an image is succeeded.
  3) Signing -> Fail: Signing an image is failed.
