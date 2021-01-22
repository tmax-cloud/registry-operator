# Environmental Variables

## The following environment variables are set to use the registry, not docker.io

|Key|Required|Description|Example|
|:---------------------------:|-----|-----------------------------------------|---|
|`IMAGE_REGISTRY`             | No  | Private image registry url including image such as `registry:2.7.1` and `tmaxcloudck/notary_server:0.6.2-rc1` etc |
|`IMAGE_REGISTRY_PULL_SECRET` | No  | Private image registry's imagepullsecret | |


## The following environment variables are for registry user.

|Key|Required|Description|Example|
|:---------------------------:|-----|-----------------------------------------|---|
|`KEYCLOAK_SERVICE`           | Yes | The URL of `Keycloak` | https://keycloak-test-service.reg-test.svc.cluster.local:8443 | 
|`CLUSTER_NAME`               | No  | If multicluster is considered, set cluster's name for distinguishing clusters. | my-kube |


## The following environment variables are for using add-ons, such as image scanning.

|Key|Required|Description|Example|
|:---------------------------:|-----|-----------------------------------------|---|
|`CLAIR_URL`                  | No  | The URL of `Clair` | http://clairsvc.default.svc.cluster.local:6060 |
|`ELASTIC_SEARCH_URL`         | No  | The URL of `Elastic Search` | http://elasticsearch-svc.default.svc.cluster.local:9200 |
|`HARBOR_NAMESPACE`           | No  | The namespace of harbor | harbor |
|`HARBOR_CORE_INGRESS`        | No  | The name of harbor core ingress  | tmax-harbor-ingress |
|`HARBOR_NOTARY_INGRESS`      | No  | The name of harbor notary ingress | tmax-harbor-ingress-notary |


## You can set the image address and imagepullsecret settings used by the operator separately.

|Key|Required|Description|Example|
|:--------------------------------:|-----|------------------------------------------|---|
|`REGISTRY_IMAGE`                  | No  | The URL of `Registry image`              | registry:2.7.1 |
|`NOTARY_SERVER_IMAGE`             | No  | The URL of `Notray server image`         | tmaxcloudck/notary_server:0.6.2-rc1 |
|`NOTARY_SIGNER_IMAGE`             | No  | The URL of `Notray signer image`         | tmaxcloudck/notary_signer:0.6.2-rc1 |
|`NOTARY_DB_IMAGE`                 | No  | The URL of `Notray db image`             | tmaxcloudck/notary_mysql:0.6.2-rc1 |
|`REGISTRY_IMAGE_PULL_SECRET`      | No  | ImagePullSecret of `Registry image`      | |
|`NOTARY_SERVER_IMAGE_PULL_SECRET` | No  | ImagePullSecret of `Notary server image` | |
|`NOTARY_SIGNER_IMAGE_PULL_SECRET` | No  | ImagePullSecret of `Notary signer image` | |
|`NOTARY_DB_IMAGE_PULL_SECRET`     | No  | ImagePullSecret of `Notary db image`     | |