# `ExternalRegistry` Usage

## What is it?

`ExternalRegistry` is a resource that register external registry. You can see a list of images in the external registry or create an ImageReplicate CR to copy images from one registry to another.

## How to create

### spec fields

|Key|Required|Type|Description|
|:-------------------------------------------:|-----|-------------------|-----|
|`spec.registryType`                          | Yes | string            | Registry type like HarborV2 |
|`spec.registryUrl`                           | Yes | string            | Registry URL (example: docker.io) |
|`spec.certificateSecret`                     | No  | string            | Certificate secret name for private registry. Secret's data key must be 'ca.crt' or 'tls.crt' |
|`spec.insecure`                              | No  | bool              | Do not verify tls certificates |
|`spec.loginId`                               | No  | string            | Login ID for registry |
|`spec.loginPassword`                         | No  | string            | Login password for registry |
|`spec.schedule`                              | No  | object            | Schedule is a cron spec for periodic sync. If you want to synchronize repository every 5 minute, enter `*/5 * * * *`. Cron spec ref: <https://ko.wikipedia.org/wiki/Cron> |

## Example

**Note**: Please check that `reg-test` namespace exists before you create the test example below. If not exists, you must create [reg-test namespace](../../config/samples/namespace.yaml).

Reference: [Test Example](../../config/samples/tmax.io_v1_externalregistry.yaml)

## Result

* State(status.state)
  * Pending: External registry is initializing.
  * NotReady: External registry is not initialized or cron job is not created.
  * Ready: External registry is synchronizing periodically by cronjob.

* State(status.state) Transition
  1) (None) -> Pending: Initializing is started.
  2) Pending -> NotReady: External registry is not initialized or cron job is not created.
  3) NotReady -> Ready: Initialized and registry cron job is operating successfully.
  4) Ready -> NotReady: registry cron job has some problems.

* Created Subresource Names in the namespace
  * RegistryCronJob: hpcd-ext-{EXTERNAL_REGISTRY_NAME}
  
  * If `spec.loginId` and `spec.loginPassword` is not empty
    (**Note**: After login secret is made, `spec.loginId` and `spec.loginPassword` will be removed for security)
    * Secret: hpcd-ext-login-{EXTERNAL_REGISTRY_NAME}
