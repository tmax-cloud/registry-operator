# Installation

## Prerequisites

* `kubectl` is installed
* `wget` is installed

## Install procedure

1. On your local machine, get released source to install.

    ```bash
    export VERSION=v0.2.1
    mkdir registry-operator-$VERSION
    wget -c https://github.com/tmax-cloud/registry-operator/archive/$VERSION.tar.gz -O - |tar -xz -C registry-operator-$VERSION --strip-components=1
    export WORKDIR=$(pwd)/registry-operator-$VERSION
    cd ${WORKDIR}
    ```

1. Create or use CA certificate

    * If you don't have a root CA certificate, excute following commandsto create new root ca certificate.

        ```bash
        cd ${WORKDIR}
        sudo chmod 755 ./config/scripts/newCertFile.sh
        ./config/scripts/newCertFile.sh
        cp ca.crt ca.key ./config/pki/
        ```

    * If you already have a root CA certificate , put the CA Certifacete in the path(./config/pki/). Each name must be `ca.crt` and `ca.key`

    * If you have to register your keycloak certificate, put the keycloak certificate in the `path(config/pki/keycloak.crt)`.

1. Set manager.yaml's configuration

    Customize env file([`config/manager/manager_config.yaml`](../config/manager/manager_config.yaml))
    * reference: [Environment Description](./envs.md)

1. Execute install.sh script

    **Note**: If you have keycloak's certificate, modify config/manager/keycloak_cert_secret.yaml file's contents

    * Create namespace, CRDs, role, etc... Then deploy the registry-operator.

        ```bash
        cd ${WORKDIR}
        sudo chmod 755 ./config/scripts/newCertSecret.sh install.sh
        ./install.sh 
        ```

1. Launch CA certificate

    * Update CA Certificate (Note: Must be applied to all nodes)
        * If the node is CentOS 7

            ```bash
            cd ${WORKDIR}
            cp ./config/pki/ca.crt /etc/pki/ca-trust/source/anchors/
            update-ca-trust
            ```

        * If the node is Ubuntu 18.04

            ```bash
            cd ${WORKDIR}
            cp ./config/pki/ca.crt /usr/local/share/ca-certificates/
            update-ca-certificates
            ```

    * Restart container runtime to apply CA (Note: Must be applied to all nodes)
        1) If container runtime is docker

            ```bash
            systemctl restart docker
            ```

        1) If container runtime is cri-o

            ```bash
            systemctl restart crio
            ```

1. Install Clair for image scanning (option)
    1) Move to resource directory and check clair config.

        ```bash
        cd config/manager/clair # then open the clair-config.yml and verify settings.
        ```

    1) Deploy server

        ```bash
        make dev
        ```

### Test your installation

* The way to verify that the registry operator works is to create a sample registry.

    ```bash
    cd ${WORKDIR}
    kubectl create -f config/samples/namespace.yaml
    kubectl create -f config/samples/tmax.io_v1_registry.yaml
    ```

    The above command creates `tmax-registry` registry in `reg-test` namespace.
    When you check the STATUS of registry with the command below, it is normal if it is creating or running.
    Otherwise, if STATUS is an empty value, the registry-operator is in startup or failed to start.

    ```bash
    kubectl get reg tmax-registry -n reg-test -w
    ```

### Tear Down

* If you want to remove all resources, execute follwing command.

    ```bash
    cd ${WORKDIR}
    chmod 755 ./uninstall.sh
    ./uninstall -a
    ```

* If you want to remove manager resources without crd resources, execute follwing command.

    ```bash
    cd ${WORKDIR}
    chmod 755 ./uninstall.sh
    ./uninstall -m
    ```

* If you want to remove only CRDs, execute follwing command.

    ```bash
    cd ${WORKDIR}
    chmod 755 ./uninstall.sh
    ./uninstall -c
    ```

* If you want to remove Clair server only, execute follwing command.

    ```bash
    cd ${WORKDIR}
    chmod 755 ./uninstall.sh
    ./uninstall -s
    ```
