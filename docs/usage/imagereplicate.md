# `ImageReplicate` Usage

## What is it?

`ImageReplicate` is a resource to copy an image between different registry.

## How to create

### spec fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.fromImage`                             | Yes | object            | Source image information |
|`spec.toImage`                               | Yes | object            | Destination image information |
|`spec.signer`                                | No  | string            | The name of the signer to sign the image you moved. This field is available only if ToImage's `RegistryType` is `HpcdRegistry` |

### spec.fromImage fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.fromImage.registryType`                | Yes | string            | Registry type like HarborV2 (Enum: HpcdRegistry;DockerHub;Docker) |
|`spec.fromImage.registryName`                | Yes | string            | metadata name of external registry or hpcd registry |
|`spec.fromImage.registryNamespace`           | Yes | string            | metadata namespace of external registry or hpcd registry |
|`spec.fromImage.image`                       | Yes | string            | Image path (example: library/alpine:3) |

### spec.toImage fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.toImage.registryType`                  | Yes | string            | Registry type like HarborV2 (Enum: HpcdRegistry;DockerHub;Docker) |
|`spec.toImage.registryName`                  | Yes | string            | metadata name of external registry or hpcd registry |
|`spec.toImage.registryNamespace`             | Yes | string            | metadata namespace of external registry or hpcd registry |
|`spec.toImage.image`                         | Yes | string            | Image path (example: library/alpine:3) |

## Example

**Note**: Please check that `reg-test` namespace exists before you create the test example below. If not exists, you must create [reg-test namespace](../../config/samples/namespace.yaml).

Reference: [Test Example](../../config/samples/tmax.io_v1_imagereplicate.yaml)

## Result

* State(status.state)
  * Pending: Initial status
  * Processing: Replicating and signing image
  * Success: Succeeded in replicating and signing image
  * Fail: Failed status while copying and signing image

* State(status.state) Transition
  1) (None) -> Pending: Initializing is started.
  2) Pending -> Processing: Replicating is stated.
  3) Processing -> Success: Replicating and signing are successfully completed.
  4) Processing -> Fail: Replicating or signing is failed

* Created Subresource Names in the namespace
  * RegistryJob: hpcd-repl-{IMAGE_REPLICATE_NAME}

  * If `spec.signer` is not empty
    * ImageSignRequest: status.imageSignRequestName
