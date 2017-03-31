include build/base.mk
ifndef SUPPRESS_DOCKER
include build/docker.mk
else
include build/deps.mk
include build/test.mk
endif
ifndef SUPPRESS_CROSSDOCK
include build/crossdock.mk
endif
ifdef TRAVIS
include build/travis.mk
endif

ifndef SUPRESS_CROSSDOCK
CI_TYPES ?= lint test examples crossdock
ifneq ($(filter crossdock,$(CI_TYPES)),)
CI_CROSSDOCK := true
CI_TYPES := $(filter-out crossdock,$(CI_TYPES))
endif
else
CI_TYPES ?= lint test examples
CI_TYPES := $(filter-out crossdock,$(CI_TYPES))
endif

CI_TYPES := $(filter-out deps,$(CI_TYPES))
ifneq ($(CI_TYPES),crossdock)
CI_TYPES := deps $(CI_TYPES)
endif

.DEFAULT_GOAL := ci

.PHONY: ci
ci: $(CI_TYPES) ## run continuous integration tasks
ifdef CI_CROSSDOCK
	$(MAKE) crossdock || ($(MAKE) crossdock-logs && false)
endif

.PHONY: help
help: ## show this help message
ifdef SUPPRESS_DOCKER
	@echo **Docker is not being used - SUPPRESS_DOCKER=$(SUPPRESS_DOCKER)**
else
	@echo **Docker is being used - SUPPRESS_DOCKER not set**
endif
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | cut -f 2,3 -d : | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
