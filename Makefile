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
endif

.PHONY: ci
ci:
	$(MAKE) $(CI_TYPES)
ifdef CI_CROSSDOCK
	$(MAKE) crossdock || $(MAKE) crossdock-logs
endif
