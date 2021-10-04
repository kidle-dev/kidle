include ../../hack/defines.mk

# Image URL to use all building/pushing image targets
IMG_KIDLECTL ?= kidledev/kidlectl
TAG?=$(shell git rev-parse --short HEAD)

##@ Development
gtest: ginkgo ## Run ginkgo tests
	ginkgo -r -v

test: fmt vet ## Run tests
	go test ./... -coverprofile cover.out

fmt: ## Run go fmt against code
	go fmt ./...

vet: ## Run go vet against code
	go vet ./...


##@ Build
build: fmt vet ## Build kidlectl binary
	$(GO_BUILD_RECIPE) -o bin/kidlectl .

run: fmt vet ## Run kidlectl
	go run ./*.go

d: docker ## -> docker
docker: docker-build docker-push ## Build and push the docker image

docker-build: build ## Build the docker image
	cd ../../; docker build --build-arg ARCH=$(ARCH) --build-arg OS=$(GOOS) . -t ${IMG_KIDLECTL}:${TAG}-${ARCH} -f cmd/kidlectl/Dockerfile

docker-push: ## Push the docker image
	docker push ${IMG_KIDLECTL}:${TAG}-${ARCH}

build-multi-arch-image: ## Multi arch docker image build
	IMAGE=${IMG_KIDLECTL} TAG=${TAG} ../../hack/push-docker-image.sh
