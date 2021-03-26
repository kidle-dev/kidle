
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
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
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Run ginkgo tests
gtest:
	ginkgo -r -v

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy-debug: manifests
	cd config/manager && kustomize edit set image controller=${IMG_DEBUG}
	kustomize build config/debug | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/api/..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build and push the docker image
docker: docker-build docker-push
d: docker

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# Build and push the docker debug image
docker-debug: docker-debug-build docker-debug-push
dd: docker-debug

# Build the docker debug image
docker-debug-build:
	docker build . -t ${IMG_DEBUG} -f Dockerfile.debug

# Push the docker debug image
docker-debug-push:
	docker push ${IMG_DEBUG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# Create a k3d registry
k3d-registry:
	k3d registry create --port 16000 kidle.localhost

# Create a k3s-kidle k8s server
k3s-create:
	k3d cluster create kidle --registry-use k3d-kidle.localhost:16000  --volume /home/nicolas/fun/kidle:/kidle --port 30123:30123@server[0]

# Delete a k3s-kidle k8s server
k3s-delete:
	k3d cluster delete kidle

# Starts k3s server
k3s-start:
	k3d cluster start kidle

# Stops k3s server
k3s-stop:
	k3d cluster stop kidle

# Write kubeconfig file
k3s-kubeconfig:
	k3d kubeconfig get kidle > kube.config

k3s-recreate: k3s-stop k3s-delete k3s-create k3s-kubeconfig
