# About Development

## Build Binary

To build manager binary execute following commands. manager binary will be made in bin directory.

```bash
cd ${WORKDIR}
make manager
```

## Build & Push Image

To build registry-operator image use operator-sdk tool. Excute following commands.

```bash
cd ${WORKDIR}
export DEV_IMG=tmaxcloudck/registry-operator:0.0.1-dev
make docker-build-dev
make docker-push-dev
```

## About CRD Generate

For Registry CRD, a field exists that requires oneOf to be used. But kubebuilder does not support it, so it needs to be modified in person. Do not create registry CRD yaml with `make manifests` command.
