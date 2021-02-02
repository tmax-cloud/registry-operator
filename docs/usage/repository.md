# `Repository` Usage

## What is it?

`Repository` is a resource including images. If you push images to registry, repository resource is automatically created.
You can see image list and delete images.

## Repository Spec

### spec fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.name`                                  | No  | string            | Repository name |
|`spec.versions`                              | No  | []ImageVersion    | Versions(=Tags) of image |
|`spec.registry`                              | No  | string            | Name of Registry which owns repository |

### spec.versions fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.versions.createdAt`                    | No  | string            | Created time of image version |
|`spec.versions.version`                      | Yes | string            | Version(=Tag) name |
|`spec.versions.delete`                       | No  | bool              | If true, this version will be deleted soon. |
|`spec.versions.signer`                       | No  | string            | If signed image, image signer name is set. |

## How to delete image

* Delete all images of the repository
  1) delete the repository resource

* Delete some images of the repository
  1) edit repository
  2) set `spec.versions.delete` field to `true`

## Result

* State(status.signerKeyState.created)
  * true: Succeeded in creating signer key
  * false: Failed to create signer key

* Created Subresource Name
  * SignerKey: {IMAGE_SIGNER_NAME}
