PWD := $(shell pwd)
UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)

CACHE_DIR := $(PWD)/.cache
TMP_DIR := $(PWD)/.tmp

BIN = $(CACHE_DIR)/$(UNAME_OS)/$(UNAME_ARCH)/bin

.PHONY: clean
clean:
	rm -rf $(TMP_DIR) $(CACHE_DIR)
