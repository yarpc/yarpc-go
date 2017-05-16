include build/base.mk
include build/dockerdeps.mk
include build/crossdockdeps.mk

.DEFAULT_GOAL := all

.PHONY: all
all: $(DOCKER) $(DOCKER_COMPOSE)
