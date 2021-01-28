## Build & Push Image
* To build registry-operator image use operator-sdk tool. Excute following commands.
    ```bash
    git clone https://github.com/tmax-cloud/registry-operator.git
	export WORKDIR=$(pwd)/registry-operator
	cd ${WORKDIR}
	export IMG=tmaxcloudck/registry-operator:0.0.1
    make docker-build
    make docker-push
    ```
