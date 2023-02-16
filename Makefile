## Tool Versions

# See https://github.com/kubernetes-sigs/kustomize for the last version
KUSTOMIZE_VERSION ?= v4@v4.5.7
# https://github.com/kubernetes-sigs/controller-tools/releases for the last version
CONTROLLER_GEN_VERSION ?= v0.8.0
# See https://pkg.go.dev/sigs.k8s.io/controller-runtime/tools/setup-envtest?tab=versions for the last version
ENVTEST_VERSION ?= v0.0.0-20221022092956-090611b34874
# See https://pkg.go.dev/golang.org/x/tools/cmd/goimports?tab=versions for the last version
OPM_VERSION ?= v1.26.2
# See github.com/operator-framework/operator-sdk/releases for the last version
OPERATOR_SDK_VERSION ?= v1.26.0

# See https://github.com/onsi/ginkgo/releases for the last version
GINKGO_VERSION ?= v1.16.5
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

# IMAGE_REGISTRY used to indicate the registery/group for the operator, bundle and catalog
IMAGE_REGISTRY ?= quay.io/medik8s
export IMAGE_REGISTRY

# When no version is set, use latest as image tags
DEFAULT_VERSION := 0.0.1
ifeq ($(origin VERSION), undefined)
IMAGE_TAG = latest
else ifeq ($(VERSION), $(DEFAULT_VERSION))
IMAGE_TAG = latest
else
IMAGE_TAG = v$(VERSION)
endif
export IMAGE_TAG

CHANNELS = candidate
export CHANNELS
DEFAULT_CHANNEL = candidate
export DEFAULT_CHANNEL

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= $(DEFAULT_VERSION)
export VERSION

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

OPERATOR_NAME ?= fence-agents-remediation

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# medik8s.io/fence-agents-remediation-bundle:$VERSION and medik8s.io/fence-agents-remediation-catalog:$VERSION.
IMAGE_TAG_BASE ?= $(IMAGE_REGISTRY)/$(OPERATOR_NAME)

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-operator-bundle:$(IMAGE_TAG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-operator-catalog:$(IMAGE_TAG)

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE)-operator:$(IMAGE_TAG)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Use kubectl, fallback to oc
KUBECTL = kubectl
ifeq (,$(shell which kubectl))
KUBECTL=oc
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

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

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: go-tidy
go-tidy: # Run go mod tidy - add missing and remove unused modules.
	go mod tidy

.PHONY: go-vendor
go-vendor:  # Run go mod vendor - make vendored copy of dependencies.
	go mod vendor

.PHONY: go-verify
go-verify: go-tidy go-vendor # Run go mod verify - verify dependencies have expected content
	go mod verify

.PHONY: test
test: manifests generate go-verify fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -i --bin-dir $(LOCALBIN) -p path)"  $(GINKGO) -v -r --keepGoing -requireSuite -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Bundle Creation Addition
## Some addition to bundle creation in the bundle

.PHONY: bundle-update
bundle-update: ## Update containerImage, and createdAt fields in the bundle's CSV
	sed -r -i "s|containerImage: .*|containerImage: $(IMG)|;" ./bundle/manifests/$(OPERATOR_NAME).clusterserviceversion.yaml
	sed -r -i "s|createdAt: .*|createdAt: `date '+%Y-%m-%d %T'`|;" ./bundle/manifests/$(OPERATOR_NAME).clusterserviceversion.yaml
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-reset-date
bundle-reset-date: ## Reset bundle's createdAt
	sed -r -i "s|createdAt: .*|createdAt: \"\"|;" ./bundle/manifests/$(OPERATOR_NAME).clusterserviceversion.yaml

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Default Tool Binaries
KUSTOMIZE_DIR ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN_DIR ?= $(LOCALBIN)/controller-gen
ENVTEST_DIR ?= $(LOCALBIN)/setup-envtest
GINKGO_DIR ?= $(LOCALBIN)/ginkgo
OPM_DIR = $(LOCALBIN)/opm
OPERATOR_SDK_DIR ?= $(LOCALBIN)/operator-sdk

## Specific Tool Binaries
KUSTOMIZE = $(KUSTOMIZE_DIR)/$(KUSTOMIZE_VERSION)/kustomize
CONTROLLER_GEN = $(CONTROLLER_GEN_DIR)/$(CONTROLLER_GEN_VERSION)/controller-gen
ENVTEST = $(ENVTEST_DIR)/$(ENVTEST_VERSION)/setup-envtest
GINKGO = $(GINKGO_DIR)/$(GINKGO_VERSION)/ginkgo
OPM = $(OPM_DIR)/$(OPM_VERSION)/opm
OPERATOR_SDK = $(OPERATOR_SDK_DIR)/$(OPERATOR_SDK_VERSION)/operator-sdk

.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-install-tool,$(KUSTOMIZE),$(KUSTOMIZE_DIR),sigs.k8s.io/kustomize/kustomize/$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-install-tool,$(CONTROLLER_GEN),$(CONTROLLER_GEN_DIR),sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION})

.PHONY: envtest ## This library helps write integration tests for your controllers by setting up and starting an instance of etcd and the Kubernetes API server, without kubelet, controller-manager or other components.
envtest: ## Download envtest-setup locally if necessary.
	$(call go-install-tool,$(ENVTEST),$(ENVTEST_DIR),sigs.k8s.io/controller-runtime/tools/setup-envtest@${ENVTEST_VERSION})

.PHONY: ginkgo
ginkgo: ## Download ginkgo locally if necessary.
	$(call go-install-tool,$(GINKGO),$(GINKGO_DIR),github.com/onsi/ginkgo/ginkgo@${GINKGO_VERSION})

# go-install-tool will delete old package $2, then 'go install' any package $3 to $1.
define go-install-tool
@[ -f $(1) ]|| { \
	set -e ;\
	rm -rf $(2) ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	go mod init tmp ;\
	BIN_DIR=$$(dirname $(1)) ;\
	mkdir -p $$BIN_DIR ;\
	echo "Downloading $(3)" ;\
	GOBIN=$$BIN_DIR GOFLAGS='' go install $(3) ;\
	rm -rf $$TMP_DIR ;\
}
endef

.PHONY: bundle
bundle: manifests operator-sdk kustomize ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(MAKE) bundle-reset-date
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-build
bundle-build: bundle-update ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
opm: ## Download opm locally if necessary.
	$(call url-install-tool, $(OPM), $(OPM_DIR),github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$${OS}-$${ARCH}-opm)

.PHONY: operator-sdk
operator-sdk: ## Download operator-sdk locally if necessary.
	$(call url-install-tool, $(OPERATOR_SDK), $(OPERATOR_SDK_DIR),github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH})

# url-install-tool will delete old package $2, then download $3 to $1.
define url-install-tool
@[ -f $(1) ]|| { \
	set -e ;\
	rm -rf $(2) ;\
	mkdir -p $(dir $(1)) ;\
	curl -sSLo $(1) $(3) ;\
	chmod +x $(1) ;\
	}
endef

.PHONY: build-tools
build-tools: ## Download & build all the tools locally if necessary.
	$(MAKE) kustomize controller-gen envtest ginkgo opm operator-sdk


# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMG) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)
