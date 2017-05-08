UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)

ifeq ($(UNAME_OS), Darwin)
  XDG_CACHE_HOME ?= $(HOME)/Library/Caches
else
  XDG_CACHE_HOME ?= $(HOME)/.cache
endif

CACHE_BASE := $(XDG_CACHE_HOME)/yarpc-go
CACHE := $(CACHE_BASE)/$(UNAME_OS)/$(UNAME_ARCH)

LIB := $(CACHE)/lib
BIN = $(CACHE)/bin

.PHONY: clean
clean: ## remove installed binaries and artifacts
	rm -rf $(CACHE_BASE)
