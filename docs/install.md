# Installation

### Prerequisites
* `kubectl` is installed

### Install procedure
1. On your local machine, clone this repository.
    ```bash
    git clone https://github.com/tmax-cloud/registry-operator.git
	export WORKDIR=$(pwd)/registry-operator
    cd ${WORKDIR}
    ```
    
2. Create or use CA certificate
	* If you don't have a root CA certificate, excute following commandsto create new root ca certificate.
		```bash
		cd ${WORKDIR}
		sudo chmod 755 ./config/scripts/newCertFile.sh
		./config/scripts/newCertFile.sh
		cp ca.crt ca.key ./config/pki/
		``` 

	* If you already have a root CA certificate , put the CA Certifacete in the path(./config/pki/). 
	Each name must be `ca.crt` and `ca.key`

	* If you have to register your keycloak certificate, put the keycloak certificate in the `path(./config/pki/keycloak.crt)`.

3. Set manager.yaml's configuration
	* Change the following export variables to the appropriate values to run.
		```bash
		cd ${WORKDIR}
		export REGISTRY_OPERATOR_VERSION=v0.1.0
		sed -i 's/{REGISTRY_OPERATOR_VERSION}/'${REGISTRY_OPERATOR_VERSION}'/g' ./config/manager/manager.yaml
		```
	* Customize env file(`config/manager/manager_config.yaml`)
		* reference: [Environment Description](./docs/envs.md) 

4. Execute install.sh script
	* Create namespace, CRDs, role, etc... Then deploy the registry-operator.
	(If you have keycloak's certificate, modify config/manager/keycloak_cert_secret.yaml file's contents)
		```bash
		cd ${WORKDIR}
		sudo chmod 755 ./config/scripts/newCertSecret.sh install.sh
		./install.sh 
		```
		
5. Launch CA certificate
	* Update CA Certificate (Note: Must be applied to all nodes)
		1) If the node is CentOS 7
		```bash
		cd ${WORKDIR}
		cp ./config/pki/ca.crt /etc/pki/ca-trust/source/anchors/
		update-ca-trust
		```

		2) If the node is Ubuntu 18.04
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

		2) If container runtime is cri-o
			```bash
			systemctl restart crio
			```

6. Install clair for image scanning (option)
	1) make secret file
		```bash
		kubectl create secret generic clairsecret --from-file=./config/manager/clair_config.yaml
		```
	2) deploy clair server
		```bash
		kubectl create -f config/manager/clair.yaml
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
    ./uninstall -c
    ```
