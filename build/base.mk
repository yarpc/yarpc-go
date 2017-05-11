UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)

XDG_CACHE_HOME ?= $(HOME)/.cache

CACHE_BASE := $(XDG_CACHE_HOME)/yarpc-go
CACHE := $(CACHE_BASE)/$(UNAME_OS)/$(UNAME_ARCH)

LIB := $(CACHE)/lib
BIN = $(CACHE)/bin

.PHONY: clean
clean: ## remove installed binaries and artifacts
	rm -rf $(CACHE_BASE)
