include etc/make/base.mk
ifndef SUPPRESS_DOCKER
include etc/make/dockerdeps.mk
include etc/make/docker.mk
else
include etc/make/deps.mk
include etc/make/local.mk
endif
ifndef SUPPRESS_CROSSDOCK
include etc/make/crossdock.mk
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
