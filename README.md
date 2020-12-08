# Overview
The registry-operator project is a service to launch private registries and to manage images in the registry on kubernetes. 

## Install
1. On your local machine, clone this repository.
    ```bash
    git clone https://github.com/tmax-cloud/registry-operator.git
    cd registry-operator
    ```
    
2. Create CA certificate
	```bash
	sudo chmod 755 ./config/scripts/newCertFile.sh
	./config/scripts/newCertFile.sh
	cp ca.crt ca.key ./config/pki/
	``` 

3. Execute install.sh script
	* Create namespace, CRDs, role, etc... Then deploy the registry-operator.
		```bash
		sudo chmod 755 ./config/scripts/newCertSecret.sh install.sh
		./install.sh 
		```
		
4. Launch CA certificate
	* Update CA Certificate (Note: Must be applied to all nodes)
		1) If the node is CentOS 7
		```bash
		cp ./config/pki/ca.crt /etc/pki/ca-trust/source/anchors/
		update-ca-trust
		```

		2) If the node is Ubuntu 18.04
		```bash
		cp ca.crt /usr/local/share/ca-certificates/
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

## [FOR DEV] Build Binary
* To build manager binary execute following commands. manager binary will be made in bin directory.
	```bash
	make manager
	```

## [FOR DEV] Build & Push Image
* To build registry-operator image use operator-sdk tool. Excute following commands.
    ```bash
	export DEV_IMG=tmaxcloudck/registry-operator:0.0.1-dev
    make docker-build-dev
    make docker-push-dev
    ```

## [FOR RELEASE] Build & Push Image
* To build registry-operator image use operator-sdk tool. Excute following commands.
    ```bash
	export IMG=tmaxcloudck/registry-operator:0.0.1
    make docker-build
    make docker-push
    ```