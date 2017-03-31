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
ci: $(CI_TYPES)
ifdef CI_CROSSDOCK
	$(MAKE) crossdock || ($(MAKE) crossdock-logs && false)
endif
