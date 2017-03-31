DOCKER_GO_VERSION ?= 1.8
DOCKERFILE := Dockerfile.$(DOCKER_GO_VERSION)
DOCKER_IMAGE := uber/yarpc-go-$(DOCKER_GO_VERSION)

DOCKER_RUN_FLAGS = -e V -e RUN
DOCKER_RUN_VOLUME_MOUNT := -v $(PWD):/go/src/go.uber.org/yarpc

.PHONY: deps
deps: ## install all dependencies
	docker build -t $(DOCKER_IMAGE) -f $(DOCKERFILE) .

.PHONY: build
build: deps ## go build all packages
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make build

.PHONY: generate
generate: deps ## call generation script
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_RUN_VOLUME_MOUNT) $(DOCKER_IMAGE) make generate

.PHONY: nogogenerate
nogogenerate: deps ## check to make sure go:generate is not used
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make nogogenerate

.PHONY: generatenodiff
generatenodiff: deps ## make sure no diff is created by make generate
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make generatenodiff

.PHONY: gofmt
gofmt: deps ## check gofmt
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make gofmt

.PHONY: govet
govet: deps ## check go vet
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make govet

.PHONY: golint
golint: deps ## check golint
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make golint

.PHONY: staticcheck
staticcheck: deps ## check staticchck
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make staticcheck

.PHONY: errcheck
errcheck: deps ## check errcheck
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make errcheck

.PHONY: verifyversion
verifyversion: deps ## verify the version in the changelog is the same as in version.go
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make verifyversion

.PHONY: lint
lint: deps ## run all linters
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make lint

.PHONY: test
test: deps ## run all tests
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make test

.PHONY: cover
cover: deps ## run all tests and output code coverage
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make cover

.PHONY: goveralls
goveralls: deps ## run code coverage and upload to coveralls
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make goveralls

.PHONY: examples
examples: deps ## run all examples tests
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make examples
