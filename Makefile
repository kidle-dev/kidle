
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


# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

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

# Restarts k3s server
k3s-restart: k3s-stop k3s-start

# Write kubeconfig file
k3s-kubeconfig:
	k3d kubeconfig get kidle > kube.config

k3s-recreate: k3s-stop k3s-delete k3s-create k3s-kubeconfig
