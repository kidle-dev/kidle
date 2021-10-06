include ../../hack/defines.mk

##@ Development
gtest: ginkgo ## Run ginkgo tests.
	ginkgo -r -v

test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...


##@ Build
build: ## Build kidlectl binary.
	$(GO_BUILD_RECIPE) -o bin/kidlectl .

run: fmt vet ## Run kidlectl.
	go run ./*.go

d: docker ## -> docker
docker: docker-build docker-push ## Build and push the docker image.

docker-build: build ## Build the docker image.
	cd ../../; docker build \
		--build-arg ARCH=${FROM_ARCH} \
		--build-arg OS=${GOOS} \
		--build-arg BUILD_REVISION=${BUILD_REVISION} \
		--build-arg BUILD_USER=${BUILD_USER} \
		--build-arg BUILD_DATE=${BUILD_DATE} \
		--build-arg BUILD_BRANCH=${BUILD_BRANCH} \
		--build-arg VERSION=${VERSION} \
		-t ${IMG_KIDLECTL}:${TAG}-${GOARCH} \
		-f cmd/kidlectl/Dockerfile \
		.

docker-push: ## Push the docker image.
	docker push ${IMG_KIDLECTL}:${TAG}-${GOARCH}

build-multi-arch-image: ## Multi arch docker image build.
	IMAGE=${IMG_KIDLECTL} TAG=${TAG} ../../hack/push-docker-image.sh
