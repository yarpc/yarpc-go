DOCKER_GO_VERSION := 1.8
DOCKERFILE := Dockerfile.$(DOCKER_GO_VERSION)
DOCKER_IMAGE := uber/yarpc-go-$(DOCKER_GO_VERSION)

DOCKER_RUN_FLAGS = \
	-e V \
	-e RUN \
	-e TRAVIS_JOB_ID \
	-e TRAVIS_PULL_REQUEST
DOCKER_RUN_VOLUME_MOUNT := -v $(PWD):/go/src/go.uber.org/yarpc

.PHONY: deps
deps:
	docker build -t $(DOCKER_IMAGE) -f $(DOCKERFILE) .

.PHONY: build
build: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make build

.PHONY: generate
generate: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_RUN_VOLUME_MOUNT) $(DOCKER_IMAGE) make generate

.PHONY: nogogenerate
nogogenerate: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make nogogenerate

.PHONY: generatenodiff
generatenodiff: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make generatenodiff

.PHONY: gofmt
gofmt: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make gofmt

.PHONY: govet
govet: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make govet

.PHONY: golint
golint: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make golint

.PHONY: staticcheck
staticcheck: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make staticcheck

.PHONY: errcheck
errcheck: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make errcheck

.PHONY: verifyversion
verifyversion: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make verifyversion

.PHONY: lint
lint: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make lint

.PHONY: test
test: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make test

.PHONY: cover
cover: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make cover

.PHONY: goveralls
goveralls: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make goveralls

.PHONY: examples
examples: deps
	docker run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE) make examples
