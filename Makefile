include build/base.mk
ifndef SUPPRESS_DOCKER
ifneq ($(shell docker version >/dev/null; echo $$?),0)
$(info -----------------------------------------------------------------)
$(info Most of our make commands run inside Docker but you do not have)
$(info Docker Daemon running. Please start the Docker Daemon and try again.)
$(info See https://docs.docker.com/engine/installation/ to get started with)
$(info Docker.)
$(info )
$(info Alternatively, set SUPPRESS_DOCKER=1 to run the command locally)
$(info against your system.)
$(error Docker is not running)
endif
include build/dockerdeps.mk
include build/docker.mk
else
include build/deps.mk
include build/local.mk
endif
ifndef SUPPRESS_CROSSDOCK
include build/crossdockdeps.mk
include build/crossdock.mk
endif
ifdef TRAVIS
include build/travis.mk
endif

CI_TYPES ?= deps lint test examples
ifndef SUPRESS_CROSSDOCK
ifneq ($(filter crossdock,$(CI_TYPES)),)
CI_CROSSDOCK := true
CI_TYPES := $(filter-out crossdock,$(CI_TYPES))
endif
else
CI_TYPES := $(filter-out crossdock,$(CI_TYPES))
endif

.DEFAULT_GOAL := ci

.PHONY: ci
ci: __print_ci $(CI_TYPES) ## run continuous integration tasks
ifdef CI_CROSSDOCK
	$(MAKE) crossdock-fresh || ($(MAKE) crossdock-logs && false)
endif

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
ifdef CI_CROSSDOCK
	$(info CI_TYPES=$(CI_TYPES) crossdock)
else
	$(info CI_TYPES=$(CI_TYPES))
endif
	@echo
