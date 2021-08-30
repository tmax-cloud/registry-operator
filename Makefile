# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
REGISTRY ?= tmaxcloudck
IMG ?= $(REGISTRY)/registry-operator:$(VERSION)
DEV_IMG ?= $(REGISTRY)/registry-operator:$(VERSION)-dev

IMG_JOB ?= $(REGISTRY)/registry-job-operator:$(VERSION)
DEV_IMG_JOB ?= $(REGISTRY)/registry-job-operator:$(VERSION)-dev

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet manager-only job-manager-only

manager-only:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/registry-operator/manager cmd/registry-operator/main.go

job-manager-only:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/registry-job-operator/manager cmd/registry-job-operator/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build:
	docker build -t ${IMG} -f images/registry-operator/Dockerfile .
	docker build -t ${IMG_JOB} -f images/registry-job-operator/Dockerfile .

# Build the docker image
docker-build-dev:
	docker build -t ${DEV_IMG} -f images/registry-operator/Dockerfile.dev .
	docker build -t ${DEV_IMG_JOB} -f images/registry-job-operator/Dockerfile.dev .

# Push the docker image
docker-push:
	docker push ${IMG}
	docker push ${IMG_JOB}

# Push the docker image
docker-push-dev:
	docker push ${DEV_IMG}
	docker push ${DEV_IMG_JOB}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Custom targets for registry operator
.PHONY: test-gen test-crd test-verify test-lint test-unit

# Test if zz_generated.deepcopy.go file is generated
test-gen: save-sha-gen generate compare-sha-gen

# Test if crd yaml files are generated
test-crd: save-sha-crd manifests compare-sha-crd

# Verify if go.sum is valid
test-verify: save-sha-mod verify compare-sha-mod

# Test code lint
test-lint:
	golangci-lint run ./... -v -E gofmt --timeout 1h0m0s

# Unit test
test-unit:
	go test -v ./pkg/...

# variable for test generate
API_V1_DIR = api/v1/
GENERATE_FILE = zz_generated.deepcopy.go

save-sha-gen:
	$(eval GENSHA=$(shell sha512sum $(API_V1_DIR)$(GENERATE_FILE)))

compare-sha-gen:
	$(eval GENSHA_AFTER=$(shell sha512sum $(API_V1_DIR)$(GENERATE_FILE)))
	@if [ "${GENSHA_AFTER}" = "${GENSHA}" ]; then echo "$(GENERATE_FILE) is not changed"; else echo "$(GENERATE_FILE) file is changed"; exit 1; fi

# variable for test crd
CRD_DIR = config/crd/bases/
CRD_1 = tmax.io_externalregistries.yaml
CRD_2 = tmax.io_imagereplicates.yaml
CRD_3 = tmax.io_imagescanrequests.yaml
CRD_4 = tmax.io_imagesigners.yaml
CRD_5 = tmax.io_imagesignrequests.yaml
CRD_6 = tmax.io_notaries.yaml
CRD_7 = tmax.io_registrycronjobs.yaml
CRD_8 = tmax.io_registryjobs.yaml
CRD_9 = tmax.io_repositories.yaml
CRD_10 = tmax.io_signerkeys.yaml


save-sha-crd:
	$(eval CRDSHA_1=$(shell sha512sum $(CRD_DIR)$(CRD_1)))
	$(eval CRDSHA_2=$(shell sha512sum $(CRD_DIR)$(CRD_2)))
	$(eval CRDSHA_3=$(shell sha512sum $(CRD_DIR)$(CRD_3)))
	$(eval CRDSHA_4=$(shell sha512sum $(CRD_DIR)$(CRD_4)))
	$(eval CRDSHA_5=$(shell sha512sum $(CRD_DIR)$(CRD_5)))
	$(eval CRDSHA_6=$(shell sha512sum $(CRD_DIR)$(CRD_6)))
	$(eval CRDSHA_7=$(shell sha512sum $(CRD_DIR)$(CRD_7)))
	$(eval CRDSHA_8=$(shell sha512sum $(CRD_DIR)$(CRD_8)))
	$(eval CRDSHA_9=$(shell sha512sum $(CRD_DIR)$(CRD_9)))
	$(eval CRDSHA_10=$(shell sha512sum $(CRD_DIR)$(CRD_10)))

compare-sha-crd:
	$(eval CRDSHA_1_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_1)))
	$(eval CRDSHA_2_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_2)))
	$(eval CRDSHA_3_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_3)))
	$(eval CRDSHA_4_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_4)))
	$(eval CRDSHA_5_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_5)))
	$(eval CRDSHA_6_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_6)))
	$(eval CRDSHA_7_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_7)))
	$(eval CRDSHA_8_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_8)))
	$(eval CRDSHA_9_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_9)))
	$(eval CRDSHA_10_AFTER=$(shell sha512sum $(CRD_DIR)$(CRD_10)))
	@if [ "${CRDSHA_1_AFTER}" = "${CRDSHA_1}" ]; then echo "$(CRD_1) is not changed"; else echo "$(CRD_1) file is changed"; exit 1; fi
	@if [ "${CRDSHA_2_AFTER}" = "${CRDSHA_2}" ]; then echo "$(CRD_2) is not changed"; else echo "$(CRD_2) file is changed"; exit 1; fi
	@if [ "${CRDSHA_3_AFTER}" = "${CRDSHA_3}" ]; then echo "$(CRD_3) is not changed"; else echo "$(CRD_3) file is changed"; exit 1; fi
	@if [ "${CRDSHA_4_AFTER}" = "${CRDSHA_4}" ]; then echo "$(CRD_4) is not changed"; else echo "$(CRD_4) file is changed"; exit 1; fi
	@if [ "${CRDSHA_5_AFTER}" = "${CRDSHA_5}" ]; then echo "$(CRD_5) is not changed"; else echo "$(CRD_5) file is changed"; exit 1; fi
	@if [ "${CRDSHA_6_AFTER}" = "${CRDSHA_6}" ]; then echo "$(CRD_6) is not changed"; else echo "$(CRD_6) file is changed"; exit 1; fi
	@if [ "${CRDSHA_7_AFTER}" = "${CRDSHA_7}" ]; then echo "$(CRD_7) is not changed"; else echo "$(CRD_7) file is changed"; exit 1; fi
	@if [ "${CRDSHA_8_AFTER}" = "${CRDSHA_8}" ]; then echo "$(CRD_8) is not changed"; else echo "$(CRD_8) file is changed"; exit 1; fi
	@if [ "${CRDSHA_9_AFTER}" = "${CRDSHA_9}" ]; then echo "$(CRD_9) is not changed"; else echo "$(CRD_9) file is changed"; exit 1; fi
	@if [ "${CRDSHA_10_AFTER}" = "${CRDSHA_10}" ]; then echo "$(CRD_10) is not changed"; else echo "$(CRD_10) file is changed"; exit 1; fi

# variable for mod
GO_MOD_FILE = go.mod
GO_SUM_FILE = go.sum

save-sha-mod:
	$(eval MODSHA=$(shell sha512sum $(GO_MOD_FILE)))
	$(eval SUMSHA=$(shell sha512sum $(GO_SUM_FILE)))

verify:
	go mod verify

compare-sha-mod:
	$(eval MODSHA_AFTER=$(shell sha512sum $(GO_MOD_FILE)))
	$(eval SUMSHA_AFTER=$(shell sha512sum $(GO_SUM_FILE)))
	@if [ "${MODSHA_AFTER}" = "${MODSHA}" ]; then echo "$(GO_MOD_FILE) is not changed"; else echo "$(GO_MOD_FILE) file is changed"; exit 1; fi
	@if [ "${SUMSHA_AFTER}" = "${SUMSHA}" ]; then echo "$(GO_SUM_FILE) is not changed"; else echo "$(GO_SUM_FILE) file is changed"; exit 1; fi

.PHONY: delete-manager delete-token delete-clair delete

delete-manager:
	-kubectl delete secret -n registry-system registry-ca
	-kubectl delete configmap -n registry-system manager-config
	-kubectl delete configmap -n registry-system registry-config
	-kubectl delete service -n registry-system registry-operator-service
	-kubectl delete deployment -n registry-system registry-job-operator
	-kubectl delete deployment -n registry-system registry-operator

delete-token:
	-kubectl delete deployment -n registry-system token-service-db
	-kubectl delete deployment -n registry-system token-service
	-kubectl delete service -n registry-system token-service-db
	-kubectl delete service -n registry-system token-service
	-kubectl delete pvc -n registry-system token-service-db-pvc
	-kubectl delete pvc -n registry-system token-service-log-pvc
	-kubectl delete ingress -n registry-system token-service-ingress
	-kubectl delete secret -n registry-system token-service

delete-clair:
	-kubectl delete rs -n registry-system clair-db
	-kubectl delete rs -n registry-system clair
	-kubectl delete service -n registry-system clair-db
	-kubectl delete service -n registry-system clair
	-kubectl delete configmap -n registry-system clair-config

delete: delete-manager delete-token delete-clair
	-kubectl delete -f config/webhook/mutating-webhook.yaml
	-kubectl delete -f config/rbac/role.yaml
	-kubectl delete -f config/rbac/role_binding.yaml
	-kubectl delete -f config/rbac/image-signer-role.yaml
	-kubectl delete -f config/apiservice/apiservice.yaml
	-kubectl delete -f config/manager/namespace.yaml

.PHONY: patch patch-job deploy-poc

patch: manager-only
	sshpass -p 'tmax@23' ssh root@172.22.11.2 rm -rf /root/go/src/github.com/tmax-cloud/registry-operator/bin/registry-operator/manager
	sshpass -p 'tmax@23' scp bin/registry-operator/manager root@172.22.11.2:/root/go/src/github.com/tmax-cloud/registry-operator/bin/registry-operator
	$(eval CURRENT_RUNNING_POD=$(shell kubectl get pod -n registry-system --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep registry-operator))
	kubectl delete po -n registry-system ${CURRENT_RUNNING_POD}

patch-job: job-manager-only
	sshpass -p 'tmax@23' ssh root@172.22.11.2 rm -rf /root/go/src/github.com/tmax-cloud/registry-operator/bin/registry-job-operator/manager
	sshpass -p 'tmax@23' scp bin/registry-job-operator/manager root@172.22.11.2:/root/go/src/github.com/tmax-cloud/registry-operator/bin/registry-job-operator
	$(eval CURRENT_RUNNING_POD=$(shell kubectl get pod -n registry-system --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep registry-job-operator))
	kubectl delete po -n registry-system ${CURRENT_RUNNING_POD}

deploy-poc:
	sshpass -p 'tmax@23' ssh root@220.90.208.100 rm -rf /root/go/src/github.com/tmax-cloud/registry-operator/bin/registry-operator/manager
	sshpass -p 'tmax@23' scp bin/registry-operator/manager root@220.90.208.100:/root/go/src/github.com/tmax-cloud/registry-operator/bin/registry-operator
