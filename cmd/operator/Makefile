include ../../hack/defines.mk

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

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

manifests: controller-gen  ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="../../..." output:crd:artifacts:config=../../config/crd/bases output:rbac:artifacts:config=../../config/rbac

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="../../hack/boilerplate.go.txt" paths="./..."


##@ Build
run: generate fmt vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config
	go run ./main.go --health-probe-bind-address=:8081 --metrics-bind-address=127.0.0.1:8080

build: generate ## Build manager binary.
	$(GO_BUILD_RECIPE) -o bin/operator main.go

d: docker ## -> docker.
docker: docker-build docker-push ## Build and push the docker image.

docker-build: build ## Build the docker image.
	cd ../../; docker build --no-cache \
		--build-arg ARCH=${FROM_ARCH} \
		--build-arg OS=${GOOS} \
		--build-arg BUILD_REVISION=${BUILD_REVISION} \
		--build-arg BUILD_USER=${BUILD_USER} \
		--build-arg BUILD_DATE=${BUILD_DATE} \
		--build-arg BUILD_BRANCH=${BUILD_BRANCH} \
		--build-arg VERSION=${VERSION} \
		-t ${IMG_OPERATOR}:${TAG}-${GOARCH} \
		-f cmd/operator/Dockerfile \
		.

docker-push: ## Push the docker image.
	docker push ${IMG_OPERATOR}:${TAG}-${GOARCH}

build-multi-arch-image: ## Multi arch docker image build.
	IMAGE=${IMG_OPERATOR} TAG=${TAG} ../../hack/push-docker-image.sh

##@ Remote debug
dd: docker-debug ## -> docker-debug
docker-debug: docker-debug-build docker-debug-push ## Build and push the docker debug image.

docker-debug-build: ## Build the docker debug image.
	docker build . -t ${IMG_DEBUG} -f Dockerfile.debug

docker-debug-push: ## Push the docker debug image.
	docker push ${IMG_DEBUG}
