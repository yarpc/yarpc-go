# Paths besides auto-detected generated files that should be excluded from
# lint results.
LINT_EXCLUDES_EXTRAS =

# Regex for 'go vet' rules to ignore
GOVET_IGNORE_RULES = \
	possible formatting directive in Error call

# List of executables needed for 'make generate'
GENERATE_DEPENDENCIES = \
	github.com/golang/mock/mockgen \
	github.com/uber/tchannel-go/thrift/thrift-gen \
	golang.org/x/tools/cmd/stringer \
	go.uber.org/thriftrw \
	go.uber.org/tools/update-license

PACKAGES := $(go list ./... | grep -v go\.uber\.org\/yarpc\/vendor)

GO_FILES := $(shell \
	find . '(' -path '*/.*' -o -path './vendor' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

# Files whose first line contains "Code generated by" are generated.
GENERATED_GO_FILES := $(shell \
	find $(GO_FILES) \
	-exec sh -c 'head -n30 {} | grep "Code generated by\|\(Autogenerated\|Automatically generated\) by\|@generated" >/dev/null' \; \
	-print)

LINT_EXCLUDES := $(GENERATED_GO_FILES) $(LINT_EXCLUDES_EXTRAS)

# Pipe lint output into this to filter out ignored files.
FILTER_LINT := grep -v $(patsubst %,-e %, $(LINT_EXCLUDES))

ERRCHECK_FLAGS ?= -ignoretests
ERRCHECK_EXCLUDES := \.Close\(\) \.Stop\(\)
FILTER_ERRCHECK := grep -v $(patsubst %,-e %, $(ERRCHECK_EXCLUDES))

CHANGELOG_VERSION = $(shell grep '^v[0-9]' CHANGELOG.md | head -n1 | cut -d' ' -f1)
INTHECODE_VERSION = $(shell perl -ne '/^const Version.*"([^"]+)".*$$/ && print "v$$1\n"' version.go)

CI_TYPES ?= lint test examples crossdock
ifneq ($(filter lint,$(CI_TYPES)),)
CI_LINT := true
endif
ifneq ($(filter test,$(CI_TYPES)),)
CI_TEST := true
endif
ifneq ($(filter cover,$(CI_TYPES)),)
CI_COVER := true
endif
ifneq ($(filter examples,$(CI_TYPES)),)
CI_EXAMPLES := true
endif
ifneq ($(filter crossdock,$(CI_TYPES)),)
CI_CROSSDOCK := true
endif
ifneq ($(filter goveralls,$(CI_TYPES)),)
CI_GOVERALLS := true
endif

CI_CACHE_DIR := $(shell pwd)/.cache
CI_DOCKER_CACHE_DIR := $(CI_CACHE_DIR)/docker
CI_DOCKER_IMAGE := yarpc_go
CI_DOCKER_CACHE_FILE := $(CI_DOCKER_CACHE_DIR)/$(CI_DOCKER_IMAGE)

TMP := $(shell pwd)/.tmp

# all we want is go get -u github.com/Masterminds/glide
# but have to pin to 0.12.3 due to https://github.com/Masterminds/glide/issues/745
GLIDE_VERSION := 0.12.3
DOCKER_COMPOSE_VERSION := 1.10.0
THRIFT_VERSION := 1.0.0-dev

BIN = $(CI_CACHE_DIR)/bin

$(BIN)/docker-compose:
	mkdir -p $(BIN)
	curl -L https://github.com/docker/compose/releases/download/$(DOCKER_COMPOSE_VERSION)/docker-compose-$(shell uname -s)-$(shell uname -m) > $(BIN)/docker-compose
	chmod +x $(BIN)/docker-compose

$(BIN)/glide:
	mkdir -p $(BIN)
	mkdir -p $(TMP)/glide
	curl -L https://github.com/Masterminds/glide/releases/download/v$(GLIDE_VERSION)/glide-v$(GLIDE_VERSION)-$(shell uname -s)-amd64.tar.gz > $(TMP)/glide/glide.tar.gz
	tar -C $(TMP)/glide -xzf $(TMP)/glide/glide.tar.gz
	mv $(TMP)/glide/$(shell uname -s | tr '[:upper:]' '[:lower:]')-amd64/glide $(BIN)/glide
	rm -rf $(TMP)/glide

$(BIN)/thrift:
	mkdir -p $(BIN)
	mkdir -p $(TMP)/thrift
	curl -L "https://github.com/uber/tchannel-go/releases/download/thrift-v$(THRIFT_VERSION)/thrift-1-$(shell uname -s)-$(shell uname -m).tar.gz" > $(TMP)/thrift/thrift.tar.gz
	tar -C $(TMP)/thrift -xzf $(TMP)/thrift/thrift.tar.gz
	mv $(TMP)/thrift/thrift-1 $(BIN)/thrift
	rm -rf $(TMP)/thrift

$(BIN)/thriftrw-plugin-yarpc: ./encoding/thrift/thriftrw-plugin-yarpc/*.go
	mkdir -p $(BIN)
	go build -o $(BIN)/thriftrw-plugin-yarpc ./encoding/thrift/thriftrw-plugin-yarpc

DOCKER_COMPOSE = $(BIN)/docker-compose
GLIDE = $(BIN)/glide
THRIFT = $(BIN)/thrift
BINS = $(DOCKER_COMPOSE) $(GLIDE) $(THRIFT) $(BIN)/thriftrw-plugin-yarpc

define generatedeprule
BINS += $(BIN)/$(shell basename $1)

$(BIN)/$(shell basename $1): vendor/$1/*.go glide.lock $(GLIDE)
	mkdir -p $(BIN)
	PATH=$(BIN):$(PATH) ./scripts/vendor-build.sh $(BIN) $1
endef

$(foreach i,$(GENERATE_DEPENDENCIES),$(eval $(call generatedeprule,$(i))))

THRIFTRW = $(BIN)/thriftrw

##############################################################################

.PHONY: lintbins
lintbins:
	@go get \
		github.com/golang/lint/golint \
		honnef.co/go/tools/cmd/staticcheck \
		github.com/kisielk/errcheck

.PHONY: coverbins
coverbins:
	@go get \
		github.com/wadey/gocovmerge \
		github.com/mattn/goveralls \
		golang.org/x/tools/cmd/cover

.PHONY: install
install: $(GLIDE)
	PATH=$(BIN):$$PATH glide install

.PHONY: clean
clean:
	rm -rf $(TMP) $(CI_CACHE_DIR)

.PHONY: build
build:
	go build $(PACKAGES)

.PHONY: generate
generate: $(BINS)
	@go get github.com/golang/mock/mockgen
	@PATH=$(BIN):$$PATH ./scripts/generate.sh

.PHONY: nogogenerate
nogogenerate:
	$(eval NOGOGENERATE_LOG := $(shell mktemp -t nogogenerate.XXXXX))
	@grep -n \/\/go:generate $(GO_FILES) 2>&1 > $(NOGOGENERATE_LOG) || true
	@[ ! -s "$(NOGOGENERATE_LOG)" ] || (echo "do not use //go:generate, add to scripts/generate.sh instead:" | cat - $(NOGOGENERATE_LOG) && false)

.PHONY: generatenodiff
generatenodiff:
	$(eval GENERATENODIFF_PRE := $(shell mktemp -t generatenodiff_pre.XXXXX))
	$(eval GENERATENODIFF_POST := $(shell mktemp -t generatenodiff_post.XXXXX))
	$(eval GENERATENODIFF_DIFF := $(shell mktemp -t generatenodiff_diff.XXXXX))
	@git status --short > $(GENERATENODIFF_PRE)
	@$(MAKE) generate
	@git status --short > $(GENERATENODIFF_POST)
	@diff $(GENERATENODIFF_PRE) $(GENERATENODIFF_POST) > $(GENERATENODIFF_DIFF) || true
	@[ ! -s "$(GENERATENODIFF_DIFF)" ] || (echo "make generate produced a diff, make sure to check these in:" | cat - $(GENERATENODIFF_DIFF) && false)


.PHONY: gofmt
gofmt:
	$(eval FMT_LOG := $(shell mktemp -t gofmt.XXXXX))
	@gofmt -e -s -l $(GO_FILES) | $(FILTER_LINT) > $(FMT_LOG) || true
	@[ ! -s "$(FMT_LOG)" ] || (echo "gofmt failed:" | cat - $(FMT_LOG) && false)

.PHONY: govet
govet:
	$(eval VET_LOG := $(shell mktemp -t govet.XXXXX))
	@go vet $(PACKAGES) 2>&1 \
		| grep -v '^exit status' \
		| grep -v "$(GOVET_IGNORE_RULES)" \
		| $(FILTER_LINT) > $(VET_LOG) || true
	@[ ! -s "$(VET_LOG)" ] || (echo "govet failed:" | cat - $(VET_LOG) && false)

.PHONY: golint
golint:
	$(eval LINT_LOG := $(shell mktemp -t golint.XXXXX))
	@cat /dev/null > $(LINT_LOG)
	@$(foreach pkg, $(PACKAGES), golint $(pkg) | $(FILTER_LINT) >> $(LINT_LOG) || true;)
	@[ ! -s "$(LINT_LOG)" ] || (echo "golint failed:" | cat - $(LINT_LOG) && false)

.PHONY: staticcheck
staticcheck:
	$(eval STATICCHECK_LOG := $(shell mktemp -t staticcheck.XXXXX))
	@staticcheck $(PACKAGES) 2>&1 | $(FILTER_LINT) > $(STATICCHECK_LOG) || true
	@[ ! -s "$(STATICCHECK_LOG)" ] || (echo "staticcheck failed:" | cat - $(STATICCHECK_LOG) && false)

.PHONY: errcheck
errcheck:
	$(eval ERRCHECK_LOG := $(shell mktemp -t errcheck.XXXXX))
	@errcheck $(ERRCHECK_FLAGS) $(PACKAGES) 2>&1 | $(FILTER_LINT) | $(FILTER_ERRCHECK) > $(ERRCHECK_LOG) || true
	@[ ! -s "$(ERRCHECK_LOG)" ] || (echo "errcheck failed:" | cat - $(ERRCHECK_LOG) && false)

.PHONY: verifyversion
verifyversion:
	@if [ "$(INTHECODE_VERSION)" = "$(CHANGELOG_VERSION)" ]; then \
		echo "yarpc-go: $(CHANGELOG_VERSION)"; \
	elif [ "$(INTHECODE_VERSION)" = "$(CHANGELOG_VERSION)-dev" ]; then \
		echo "yarpc-go (development): $(INTHECODE_VERSION)"; \
	else \
		echo "Version number in version.go does not match CHANGELOG.md"; \
		echo "version.go: $(INTHECODE_VERSION)"; \
		echo "CHANGELOG : $(CHANGELOG_VERSION)"; \
		exit 1; \
	fi

.PHONY: lint
lint: lintbins generatenodiff nogogenerate gofmt govet golint staticcheck errcheck verifyversion

.PHONY: test
test: $(THRIFTRW)
	PATH=$(BIN):$$PATH go test -race $(PACKAGES)


.PHONY: cover
cover: coverbins $(THRIFTRW)
	PATH=$(BIN):$$PATH ./scripts/cover.sh $(shell go list $(PACKAGES))
	go tool cover -html=cover.out -o cover.html

.PHONY: goveralls
goveralls:
	goveralls -coverprofile=cover.out -service=travis-ci

.PHONY: examples
examples:
	$(MAKE) -C internal/examples

.PHONY: crossdock
crossdock: $(DOCKER_COMPOSE)
	PATH=$(BIN):$$PATH docker-compose kill go
	PATH=$(BIN):$$PATH docker-compose rm -f go
	PATH=$(BIN):$$PATH docker-compose build go
	PATH=$(BIN):$$PATH docker-compose run crossdock


.PHONY: crossdock-fresh
crossdock-fresh: $(DOCKER_COMPOSE)
	PATH=$(BIN):$$PATH docker-compose kill
	PATH=$(BIN):$$PATH docker-compose rm --force
	PATH=$(BIN):$$PATH docker-compose pull
	PATH=$(BIN):$$PATH docker-compose build
	PATH=$(BIN):$$PATH docker-compose run crossdock

.PHONY: crossdock-logs
crossdock-logs: $(DOCKER_COMPOSE)
	PATH=$(BIN):$$PATH docker-compose logs

.PHONY: ci-docker-load
ci-docker-load:
ifdef CI_CROSSDOCK
	if [ -f $(CI_DOCKER_CACHE_FILE) ]; then gunzip -c $(CI_DOCKER_CACHE_FILE) | docker load; fi
endif

.PHONY: ci-docker-save
ci-docker-save:
ifdef CI_CROSSDOCK
	mkdir -p $(CI_DOCKER_CACHE_DIR)
	docker save $(shell docker history -q $(CI_DOCKER_IMAGE) | grep -v '<missing>') | gzip > $(CI_DOCKER_CACHE_FILE)
endif

.PHONY: ci
ci:
	@echo Running $(CI_TYPES)
ifdef CI_LINT
	@$(MAKE) lint
endif
ifdef CI_TEST
	@$(MAKE) test
endif
ifdef CI_COVER
	@$(MAKE) cover
endif
ifdef CI_GOVERALLS
	@$(MAKE) goveralls
endif
ifdef CI_EXAMPLES
	@$(MAKE) examples
endif
ifdef CI_CROSSDOCK
	@$(MAKE) crossdock || $(MAKE) crossdock-logs
endif

.PHONY: travis-docker-push
travis-docker-push:
ifdef CI_TRAVIS_DOCKER_PUSH
	./scripts/travis-docker-push.sh
endif
