
# Image URL to use all building/pushing image targets
IMG ?= kidlectl:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: kidlectl

# Run tests
test: fmt vet
	go test ./... -coverprofile cover.out

# Run ginkgo tests
gtest: ginkgo
	ginkgo -r -v

# Build manager binary
kidlectl: fmt vet
	go build -o bin/kidlectl .

# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet
	go run ./main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

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
