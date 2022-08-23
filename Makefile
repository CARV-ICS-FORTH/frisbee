# go options
GO                 ?= go
LDFLAGS            :=
GOFLAGS            :=

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# VERSION defines the project version for the operator.
# Update this value when you upgrade the version of your project.
FrisbeeVersion=$(shell cat VERSION)
FRISBEE_NAMESPACE := frisbee

CRD_DIR ?=	charts/platform/crds
WEBHOOK_DIR ?= charts/platform/templates/webhook
RBAC_DIR ?= charts/platform/templates/rbac
CERTS_DIR ?= /tmp/k8s-webhook-server/serving-certs

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# frisbee.dev/frisbee-bundle:$VERSION and frisbee.dev/frisbee-catalog:$VERSION.
IMAGE_TAG_BASE ?= icsforth

# Image URL to use all building/pushing image targets
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
IMG ?= $(IMAGE_TAG_BASE)/frisbee-operator:$(FrisbeeVersion)

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.21


# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# Dependencies on external binaries

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@latest)

ENVTEST = $(shell pwd)/bin/setup-envtest
envtest:
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)


# Find or download gen-crd-api-reference-docs
API_GEN = $(shell pwd)/bin/gen-crd-api-reference-docs
gen-crd-api-reference-docs:
	$(call go-get-tool,$(API_GEN),github.com/ahmetb/gen-crd-api-reference-docs@v0.3.0)


##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.DEFAULT_GOAL := help

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
# CRD_OPTIONS ?= "crd:trivialVersions=true" # "crd:trivialVersions=true,preserveUnknownFields=false"
CRD_OPTIONS ?= crd

##@ Development
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..."  output:crd:artifacts:config=${CRD_DIR}
	#$(CONTROLLER_GEN) webhook paths="./..."  output:webhook:artifacts:config=${WEBHOOK_DIR}
	$(CONTROLLER_GEN) rbac:roleName=frisbee paths="./..."  output:rbac:artifacts:config=${RBAC_DIR}

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

test: generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out


##@ Documentation

api-docs: gen-crd-api-reference-docs	## Generate API reference documentation
	$(API_GEN) -api-dir=./api/v1alpha1 -config=./hack/api-docs/config.json -template-dir=./hack/api-docs/template -out-file=./docs/api.html -v=10


##@ Build

.PHONY: certs
certs:  ## Download certs under 'certs' folder
	@echo "===> Download Certs <==="
	@echo "CertDir ${CERTS_DIR}"
	@mkdir -p ${CERTS_DIR}
	@kubectl get secret webhook-tls -n ${FRISBEE_NAMESPACE} -o json | jq -r '.data["tls.key"]' | base64 -d > ${CERTS_DIR}/tls.key
	@kubectl get secret webhook-tls -n ${FRISBEE_NAMESPACE} -o json | jq -r '.data["tls.crt"]' | base64 -d > ${CERTS_DIR}/tls.crt

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/manager/main.go

run: generate fmt vet certs ## Run a controller from your host.
	@echo "===> Run Frisbee Controller on Namespace ${FRISBEE_NAMESPACE} <==="
	go run -race ./cmd/manager/main.go -cert-dir=${CERTS_DIR}

docker-build: test ## Build docker image for the Frisbee controller.
	@echo "===> Build Frisbee Container <==="
	docker build -t ${IMG} .

docker-run: docker-build ## Build and Run docker image for the Frisbee controller.
	@echo "===> Run Frisbee Container Locally <==="

	# --rm automatically clean up the container when the container exits
	# -ti allocate a pseudo-TTY and ieep STDIN open even if not attached
	# -v mount the local kubernetes configuration to the container
	docker run --rm -ti -v ${HOME}/.kube/:/home/default/.kube ${IMG}


##@ Deployment
docker-push: docker-build ## Push the latest docker image for Frisbee controller.
	@echo "===> Tag ${IMG} as frisbee-operator:latest <==="
	docker tag ${IMG} $(IMAGE_TAG_BASE)/frisbee-operator:latest
	@echo "===> Push frisbee:operator:latest <==="
	docker push $(IMAGE_TAG_BASE)/frisbee-operator:latest


install: generate ## Deploy platform to the K8s cluster specified in ~/.kube/config.
	echo ">> yamllint charts/platform/ | grep -v \"line too long\""
	echo ">> helm upgrade --install my-frisbee charts/platform"

uninstall: ## Undeploy platform from the K8s cluster specified in ~/.kube/config.
	echo ">> helm uninstall my-frisbee"


release: ## Release a new version of Frisbee.
	if [[ -z "${FrisbeeVersion}" ]]; then echo "VERSION is not set"; exit 1; fi
	echo "${FrisbeeVersion}" > VERSION
	git add VERSION
	git commit -m "Bump version"
	git tag ${FrisbeeVersion}
	# git push --set-upstream origin $(git branch --show-current) && git push --tags

