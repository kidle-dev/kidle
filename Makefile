
WHAT ?= operator,kidlectl


all:
	hack/make-rules/build.sh all $(WHAT)

run:
	hack/make-rules/build.sh run $(WHAT)

test:
	hack/make-rules/build.sh test $(WHAT)

gtest:
	hack/make-rules/build.sh gtest $(WHAT)

build:
	hack/make-rules/build.sh build $(WHAT)

docker:
	hack/make-rules/build.sh docker $(WHAT)

d:
	hack/make-rules/build.sh d $(WHAT)

docker-build:
	hack/make-rules/build.sh docker-build $(WHAT)

docker-push:
	hack/make-rules/build.sh docker-push $(WHAT)

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG_OPERATOR}
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
