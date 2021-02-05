# About Release

## How To Release

We release it through github action. The action would be executed by publishing release. The release is made by [release draft](https://github.com/tmax-cloud/registry-operator/releases).

The release note includes in changelog like what is featured or modified.

## Build & Push Image

**NOTE**: If you are deploying an image manually without releasing it with action, you must also handle changelog and tag manually. We recommend releasing via github action

To build registry-operator image use operator-sdk tool. Excute following commands.

```bash
git clone https://github.com/tmax-cloud/registry-operator.git
export WORKDIR=$(pwd)/registry-operator
cd ${WORKDIR}
export IMG=tmaxcloudck/registry-operator:0.0.1
make docker-build
make docker-push
```
