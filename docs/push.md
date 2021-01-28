# How to push image to private registry

1. Trust Self Signed CA Certificate
    * (If your registry is registry made by registry-operator in hypercloud, there is CA Certificate in `hpcd-registry-rootca` secret of registry's namespace. `ca.crt` in `hpcd-registry-rootca` secret is the self signed ca certificate.)

    * Move `ca.crt` to CA certificate directory.
        1) If the node is CentOS 7
        ```bash
        cp ca.crt /etc/pki/ca-trust/source/anchors/
		update-ca-trust
        ```

		2) If the node is Ubuntu 18.04
		```bash
		cp ca.crt /usr/local/share/ca-certificates/
		update-ca-certificates
		```

2. Login Registry
    ```bash
    export REGISTRY_URL={REGISTRY_URL}
    docker login ${REGISTRY_URL}        # enter username and password
    ```

    * Example
    ```bash
    export REGISTRY_URL=192.168.6.100
    docker login ${REGISTRY_URL}
    ```

3. Push Image
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