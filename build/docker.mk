DOCKER_GO_VERSION ?= 1.8
DOCKERFILE := Dockerfile.$(DOCKER_GO_VERSION)
DOCKER_IMAGE := uber/yarpc-go-$(DOCKER_GO_VERSION)

ifdef DOCKER_HOST
DOCKER_BUILD_FLAGS ?= --compress
endif
DOCKER_RUN_FLAGS ?= -e V -e RUN -e EXAMPLES_JOBS -e WITHIN_DOCKER=1
DOCKER_VOLUME_FLAGS=-v $(shell pwd):/go/src/go.uber.org/yarpc

.PHONY: deps
deps: $(DOCKER) ## install all dependencies
	PATH=$$PATH:$(BIN) docker build $(DOCKER_BUILD_FLAGS) -t $(DOCKER_IMAGE) -f $(DOCKERFILE) .

.PHONY: build
build: deps ## go build all packages
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make build

.PHONY: generate
generate: deps ## call generation script
	PATH=$$PATH:$(BIN) docker run $(DOCKER_VOLUME_FLAGS) $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make generate

.PHONY: nogogenerate
nogogenerate: deps ## check to make sure go:generate is not used
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make nogogenerate

.PHONY: generatenodiff
generatenodiff: deps ## make sure no diff is created by make generate
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make generatenodiff

.PHONY: gofmt
gofmt: deps ## check gofmt
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make gofmt

.PHONY: govet
govet: deps ## check go vet
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make govet

.PHONY: golint
golint: deps ## check golint
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make golint

.PHONY: staticcheck
staticcheck: deps ## check staticchck
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make staticcheck

.PHONY: errcheck
errcheck: deps ## check errcheck
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make errcheck

.PHONY: verifycodecovignores
verifycodecovignores: deps ## check verifycodecovignores
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make verifycodecovignores

.PHONY: verifyversion
verifyversion: deps ## verify the version in the changelog is the same as in version.go
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make verifyversion

.PHONY: lint
lint: deps ## run all linters
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make lint

.PHONY: test
test: deps ## run all tests
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make test

.PHONY: cover
cover: deps ## run all tests and output code coverage
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make cover

.PHONY: codecov
codecov: deps ## run code coverage and upload to coveralls
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make codecov

.PHONY: examples
examples: deps ## run all examples tests
	PATH=$$PATH:$(BIN) docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make examples

.PHONY: shell
shell: deps ## go into a bash shell in docker with the repository linked as a volume
	PATH=$$PATH:$(BIN) docker run -it $(DOCKER_VOLUME_FLAGS) $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) /bin/bash
