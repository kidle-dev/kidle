# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Force bash as shell
SHELL := /bin/bash

# Image URL to use all building/pushing image targets
IMG_OPERATOR ?= kidledev/kidle-operator
IMG_KIDLECTL ?= kidledev/kidlectl

# Defines some commons environment variables
PROJECT_DIR := $(shell dirname $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST)))))
BIN_DIR := $(PROJECT_DIR)/bin
ENVTEST_ASSETS_DIR=$(BIN_DIR)/test
TAG?=$(shell git rev-parse --short HEAD)

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(BIN_DIR) go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef


GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
ifeq ($(GOARCH),arm)
	FROM_ARCH=armv7
else
	FROM_ARCH=$(GOARCH)
endif

VERSION?=$(shell cat $(PROJECT_DIR)/VERSION | tr -d " \t\n\r")
BUILD_DATE=$(shell date +"%Y%m%d-%T")
# source: https://docs.github.com/en/free-pro-team@latest/actions/reference/environment-variables#default-environment-variables
ifndef GITHUB_ACTIONS
	BUILD_USER?=$(USER)
	BUILD_BRANCH?=$(shell git branch --show-current)
	BUILD_REVISION?=$(shell git rev-parse --short HEAD)
else
	BUILD_USER=Action-Run-ID-$(GITHUB_RUN_ID)
	BUILD_BRANCH=$(GITHUB_REF:refs/heads/%=%)
	BUILD_REVISION=$(GITHUB_SHA)
endif

KIDLE_VERSION_PKG=github.com/kidle-dev/kidle/pkg

# The ldflags for the go build process to set the version related data.
GO_BUILD_LDFLAGS=\
	-s \
	-X $(KIDLE_VERSION_PKG)/version.Revision=$(BUILD_REVISION)  \
	-X $(KIDLE_VERSION_PKG)/version.BuildUser=$(BUILD_USER) \
	-X $(KIDLE_VERSION_PKG)/version.BuildDate=$(BUILD_DATE) \
	-X $(KIDLE_VERSION_PKG)/version.Branch=$(BUILD_BRANCH) \
	-X $(KIDLE_VERSION_PKG)/version.Version=$(VERSION)

GO_BUILD_RECIPE=\
	env GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	CGO_ENABLED=0 \
	go build -ldflags="$(GO_BUILD_LDFLAGS)"

##@ Common commands
KUSTOMIZE = $(BIN_DIR)/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

CONTROLLER_GEN = $(BIN_DIR)/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

GINKGO = $(BIN_DIR)/ginkgo
ginkgo: ## Download ginkgo locally if necessary.
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo@v1.16.4)

GOLANGCI_LINT = $(BIN_DIR)/golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1)

REVIVE = $(BIN_DIR)/revive
revive-install: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(REVIVE),github.com/mgechev/revive@v1.1.2)

GIT_CHGLOG = $(BIN_DIR)/git-chglog
git-chglog: ## Download git-chglog locally if necessary.
	$(call go-get-tool,$(GIT_CHGLOG),github.com/git-chglog/git-chglog/cmd/git-chglog@v0.15.0)


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

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

env: ## Display Makefile environment variables
	@echo IMG_OPERATOR=$(IMG_OPERATOR)
	@echo IMG_KIDLECTL=$(IMG_KIDLECTL)
	@echo TAG=$(TAG)
	@echo GOBIN=$(GOBIN)
	@echo GOARCH=$(GOARCH)
	@echo SHELL=$(SHELL)
	@echo PROJECT_DIR=$(PROJECT_DIR)
	@echo BIN_DIR=$(BIN_DIR)
	@echo ENVTEST_ASSETS_DIR=$(ENVTEST_ASSETS_DIR)
	@echo KUSTOMIZE=$(KUSTOMIZE)
	@echo CONTROLLER_GEN=$(CONTROLLER_GEN)
	@echo GINKGO=$(GINKGO)
	@echo GOLANGCI_LINT=$(GOLANGCI_LINT)
