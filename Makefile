include etc/make/base.mk
ifndef SUPPRESS_DOCKER
include etc/make/dockerdeps.mk
include etc/make/docker.mk
else
include etc/make/deps.mk
include etc/make/local.mk
endif

CI_TYPES ?= lint test examples

.DEFAULT_GOAL := ci

.PHONY: ci
ci: __print_ci $(CI_TYPES) ## run continuous integration tasks

.PHONY: help
help: __print_info ## show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | cut -f 2,3 -d : | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: __print_info
__print_info:
ifdef SUPPRESS_DOCKER
	$(info Docker is not being used)
else
	$(info Docker is being used)
ifdef DOCKER_HOST
	$(info DOCKER_HOST=$(DOCKER_HOST))
endif
	$(info DOCKER_GO_VERSION=$(DOCKER_GO_VERSION))
	$(info DOCKER_BUILD_FLAGS=$(DOCKER_BUILD_FLAGS))
	$(info DOCKER_RUN_FLAGS=$(DOCKER_RUN_FLAGS))
endif
	@echo

.PHONY: __print_ci
__print_ci: __print_info
	$(info CI_TYPES=$(CI_TYPES))
	@echo
