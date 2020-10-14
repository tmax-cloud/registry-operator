# Overview
The registry-operator project is a service to launch private registries and to manage images in the registry on kubernetes. 

## Install
1. On your local machine, clone this repository.
    ```bash
    $ git clone https://github.com/tmax-cloud/registry-operator.git
    $ cd registry-operator
    ```
    
2. Create CA certificate
	```bash
	$ sudo chmod 755 ./scripts/newCertFile.sh
	$ ./scripts/newCertFile.sh
	$ cp ca.crt ca.key ./deploy/pki/
	``` 

3. Execute install.sh script
	* Create namespace, CRDs, role, etc... Then deploy the registry-operator.
		```bash
		$ sudo chmod 755 install.sh
		$ ./install.sh 
		```
		
4. Launch CA certificate
	* Update CA Certificate (Note: Must be applied to all nodes)
		1) If the node is CentOS 7
		```bash
		$ cp ./deploy/pki/ca.crt /etc/pki/ca-trust/source/anchors/
		$ update-ca-trust
		```

		2) If the node is Ubuntu 18.04
		```bash
		$ cp ca.crt /usr/local/share/ca-certificates/
		$ update-ca-certificates
		```
		
	* Restart container runtime to apply CA (Note: Must be applied to all nodes)
		1) If container runtime is docker
			```bash
			$ systemctl restart docker
			```

		2) If container runtime is cri-o
			```bash
			$ systemctl restart crio
			```

## Build Image
* To build registry-operator image use operator-sdk tool.
    ```bash
    $ operator-sdk build registry-operator:nightly
    ```

* Note
	* If "/bin/sh: /usr/local/bin/user_setup: Permission denied" occerred, execute below command.
		 ```bash
		$ sudo chmod 755 build/bin/*
		```

	* If "/bin/sh: /usr/local/bin/user_setup: /bin/sh^M: bad interpreter: No such file or directory" (^M is `ctrl + v + m`) occerred, execute below command.
		```bash
		$ sed -i 's/^M//g' build/bin/user_setup
		$ sed -i 's/^M//g' build/bin/entrypoint
		```

