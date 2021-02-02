# `ImageSigner` Usage

## What is it?

`ImageSigner` is a subject to sign images in registry. It is a cluster scope resource. All users can get other ImageSigner but cannot edit the resource.
ImageSigner

## How to create

### spec fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.name`                                  | No  | string            | ImageSigner's name |
|`spec.email`                                 | No  | string            | ImageSigner's email |
|`spec.phone`                                 | No  | string            | ImageSigner's phone number |
|`spec.team`                                  | No  | string            | ImageSigner's team |
|`spec.description`                           | No  | string            | Additional information of ImageSigner |

## Example

Reference: [Test Example](../../config/samples/tmax.io_v1_imagesigner.yaml)

## Result

* State(status.signerKeyState.created)
  * true: Succeeded in creating signer key
  * false: Failed to create signer key

* Created Subresource Name
  * SignerKey: {IMAGE_SIGNER_NAME}
  