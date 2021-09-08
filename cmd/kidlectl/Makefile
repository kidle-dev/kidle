include ../../hack/defines.mk

# Image URL to use all building/pushing image targets
IMG_KIDLECTL ?= kidle/kidlectl:latest


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
	go build -o bin/kidlectl .

run: fmt vet ## Run kidlectl
	go run ./main.go

d: docker ## -> docker
docker: docker-build docker-push ## Build and push the docker image

docker-build: test ## Build the docker image
	cd ../../; docker build . -t ${IMG_KIDLECTL} -f cmd/kidlectl/Dockerfile

docker-push: ## Push the docker image
	docker push ${IMG_KIDLECTL}
