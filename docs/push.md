# How to push image to private registry

1. Trust Self Signed CA Certificate

    * (If your registry is registry made by registry-operator in hypercloud, there is CA Certificate in `hpcd-registry-rootca` secret of registry's namespace. `ca.crt` in `hpcd-registry-rootca` secret is the self signed ca certificate.)

        ```bash
        # Command to get rootca in default namespace and create 'ca.crt' file
        export NAMESPACE=default
        kubectl get secret hpcd-registry-rootca -n ${NAMESPACE} -o="jsonpath={.data['ca\.crt']}" |base64 -d > ca.crt
        ```

    * Move `ca.crt` to CA certificate directory.

        * If the node is CentOS 7

            ```bash
            cp ca.crt /etc/pki/ca-trust/source/anchors/
            update-ca-trust
            ```

        * If the node is Ubuntu 18.04

            ```bash
            cp ca.crt /usr/local/share/ca-certificates/
            update-ca-certificates
            ```

1. Docker Restart

    ```bash
    systemctl restart docker
    ```

1. Login Registry

**Note**: If you don't know registry's username and password, refer to [Get Registry Login Username Password from ImagePullSecret](#get-registry-login-username-and-password-from-imagepullsecret)

```bash
export REGISTRY_URL={REGISTRY_URL}
docker login ${REGISTRY_URL}        # Enter username and password.
```

* Example

    ```bash
    export REGISTRY_URL=192.168.6.100
    docker login ${REGISTRY_URL}
    ```

1. Push Image

    ```bash
    export IMAGE={IMAGE}
    docker pull ${IMAGE}
    docker tag ${IMAGE} ${REGISTRY_URL}/${IMAGE}
    docker push ${REGISTRY_URL}/${IMAGE}
    ```

    * Example

        ```bash
        export IMAGE=nginx:latest
        docker pull ${IMAGE}
        docker tag ${IMAGE} ${REGISTRY_URL}/${IMAGE}
        docker push ${REGISTRY_URL}/${IMAGE}
        ```

## Get Registry Login Username and Password from ImagePullSecret

```bash
export REGISTRY={REGISTRY_URL}
export IMAGE_PULL_SECRET={IMAGE_PULL_SECRET}
export NAMESPACE={IMAGE_PULL_SECRET_NAMESPACE}
kubectl get secret ${IMAGE_PULL_SECRET} -n ${NAMESPACE} -o="jsonpath={.data['\.dockerconfigjson']}" |base64 -d |jq -r .auths.\"$REGISTRY\".auth |base64 -d
 ```

* Example:

    ```bash
    export REGISTRY=192.168.6.100
    export IMAGE_PULL_SECRET=hpcd-registry-tmax-registry
    export NAMESPACE=default
    kubectl get secret ${IMAGE_PULL_SECRET} -n ${NAMESPACE} -o="jsonpath={.data['\.dockerconfigjson']}" |base64 -d |jq -r .auths.\"$REGISTRY\".auth |base64 -d
    ```

    The result is username and password like `username:password`
