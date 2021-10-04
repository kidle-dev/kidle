include hack/defines.mk

WHAT ?= operator,kidlectl


##@ Development
gtest: ## Run ginkgo tests of the $WHAT target.
	hack/make-rules/build.sh gtest $(WHAT)

test: ## Run go tests of the $WHAT target.
	hack/make-rules/build.sh test $(WHAT)

manifests:  ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	cd cmd/operator && make manifests

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	cd cmd/operator && make generate

lint: golangci-lint ## Run golangci-lint globally
	$(GOLANGCI_LINT) run

##@ Build
run: ## Run the $WHAT target
	hack/make-rules/build.sh run $(WHAT)

build: ## Build the $WHAT target.
	hack/make-rules/build.sh build $(WHAT)

d: docker ## -> docker
docker: ## Build and push the docker image of the $WHAT target.
	hack/make-rules/build.sh docker $(WHAT)

docker-build: ## Build docker image of the $WHAT target.
	hack/make-rules/build.sh docker-build $(WHAT)

docker-push: ## Push docker image of the $WHAT target.
	hack/make-rules/build.sh docker-push $(WHAT)


##@ Deployment
install: kustomize manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: kustomize manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: kustomize manifests ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG_OPERATOR}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

deploy-debug: kustomize manifests ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG_DEBUG}
	$(KUSTOMIZE) build config/debug | kubectl apply -f -


##@ Local kubernetes environment
k3d-registry: ## Create a k3d registry.
	k3d registry create --port 16000 kidle.localhost

k3s-create: ## Create a k3s-kidle k8s server.
	k3d cluster create kidle --registry-use k3d-kidle.localhost:16000  --volume /home/nicolas/fun/kidle:/kidle --port 30123:30123@server[0]

k3s-delete: ## Delete a k3s-kidle k8s server.
	k3d cluster delete kidle

k3s-start: ## Starts k3s server.
	k3d cluster start kidle

k3s-stop: ## Stops k3s server.
	k3d cluster stop kidle

k3s-restart: k3s-stop k3s-start ## Restarts k3s server.

k3s-kubeconfig: ## Write kubeconfig file.
	k3d kubeconfig get kidle > kube.config

k3s-recreate: k3s-stop k3s-delete k3s-create k3s-kubeconfig ## Recreate a k3s-kidle k8s server.
