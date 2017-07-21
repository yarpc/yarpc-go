UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)

ifeq ($(UNAME_OS),Darwin)
  XDG_CACHE_HOME ?= $(HOME)/Library/Caches
else
  XDG_CACHE_HOME ?= $(HOME)/.cache
endif

CACHE_BASE := $(XDG_CACHE_HOME)/yarpc-go
CACHE := $(CACHE_BASE)/$(UNAME_OS)/$(UNAME_ARCH)

LIB := $(CACHE)/lib
BIN = $(CACHE)/bin

DOCKER_COMPOSE_VERSION := 1.11.2
DOCKER_COMPOSE_OS := $(UNAME_OS)
DOCKER_COMPOSE_ARCH := $(UNAME_ARCH)

DOCKER_COMPOSE_LIB = $(LIB)/docker-compose-$(DOCKER_COMPOSE_VERSION)
DOCKER_COMPOSE_BIN = $(DOCKER_COMPOSE_LIB)/docker-compose
DOCKER_COMPOSE = $(BIN)/docker-compose

$(DOCKER_COMPOSE_BIN):
	@mkdir -p $(DOCKER_COMPOSE_LIB)
	curl -L "https://github.com/docker/compose/releases/download/$(DOCKER_COMPOSE_VERSION)/docker-compose-$(DOCKER_COMPOSE_OS)-$(DOCKER_COMPOSE_ARCH)" > $(DOCKER_COMPOSE_BIN)

$(DOCKER_COMPOSE): $(DOCKER_COMPOSE_BIN)
	@mkdir -p $(BIN)
	cp $(DOCKER_COMPOSE_BIN) $(DOCKER_COMPOSE)
	@chmod +x $(DOCKER_COMPOSE)

.PHONY: clean
clean: ## remove installed binaries and artifacts
	rm -rf $(CACHE_BASE)

.PHONY: compose-codecov
compose-codecov: SHELL := /bin/bash
compose-codecov: $(DOCKER_COMPOSE) ## run code coverage and upload to coveralls inside docker-compose
	$(DOCKER_COMPOSE) kill
	$(DOCKER_COMPOSE) rm --force
	$(DOCKER_COMPOSE) build gotest
	$(DOCKER_COMPOSE) run $(shell bash <(curl -s https://codecov.io/env)) $(DOCKER_RUN_FLAGS) gotest make codecov

.PHONY: compose-cover
compose-cover: $(DOCKER_COMPOSE) ## run all tests and output code coverage inside docker-compose
	$(DOCKER_COMPOSE) kill
	$(DOCKER_COMPOSE) rm --force
	$(DOCKER_COMPOSE) build gotest
	$(DOCKER_COMPOSE) run $(DOCKER_RUN_FLAGS) gotest make cover

.PHONY: compose-test
compose-test: $(DOCKER_COMPOSE) ## run all tests inside docker-compose
	$(DOCKER_COMPOSE) kill
	$(DOCKER_COMPOSE) rm --force
	$(DOCKER_COMPOSE) build gotest
	$(DOCKER_COMPOSE) run $(DOCKER_RUN_FLAGS) gotest make test
