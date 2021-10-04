include ../../hack/defines.mk

# Image URL to use all building/pushing image targets
IMG_OPERATOR ?= kidledev/kidle-operator

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"


##@ Development
gtest: manifests generate ginkgo  ## Run ginkgo tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); $(GINKGO) -r -v ../../pkg

test: manifests generate fmt vet ## Run go tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

fmt: ## Run go fmt against code
	go fmt ./...

vet: ## Run go vet against code
	go vet ./...

manifests: controller-gen  ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="../../pkg/api/..." output:crd:artifacts:config=../../config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="../../hack/boilerplate.go.txt" paths="./..."


##@ Build
run: generate fmt vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config
	go run ./main.go

build: generate fmt vet ## Build manager binary.
	go build -o bin/operator main.go

d: docker ## -> docker.
docker: docker-build docker-push ## Build and push the docker image.

docker-build: build ## Build the docker image.
	cd ../../; docker build --build-arg ARCH=$(ARCH) --build-arg OS=$(GOOS) -t ${IMG_OPERATOR}:${TAG}-${ARCH} -f cmd/operator/Dockerfile .

docker-push: ## Push the docker image.
	docker push ${IMG_OPERATOR}:${TAG}-${ARCH}

build-multi-arch-image: ## Multi arch docker image build
	IMAGE=${IMG_KIDLECTL} TAG=${TAG} ../../hack/push-docker-image.sh

##@ Remote debug
dd: docker-debug ## -> docker-debug
docker-debug: docker-debug-build docker-debug-push ## Build and push the docker debug image.

docker-debug-build: ## Build the docker debug image.
	docker build . -t ${IMG_DEBUG} -f Dockerfile.debug

docker-debug-push: ## Push the docker debug image.
	docker push ${IMG_DEBUG}
