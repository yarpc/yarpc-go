GO_VERSION := $(shell go version | awk '{ print $$3 }')
GO_MINOR_VERSION := $(word 2,$(subst ., ,$(GO_VERSION)))
LINTABLE_MINOR_VERSIONS := 8
ifneq ($(filter $(LINTABLE_MINOR_VERSIONS),$(GO_MINOR_VERSION)),)
SHOULD_LINT := true
endif

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

##############################################################################

export GO15VENDOREXPERIMENT=1

PACKAGES := $(shell glide novendor)

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

DOCKER_IMAGE := yarpc/yarpc-go:latest

DOCKER_COMPOSE_VERSION := 1.10.0

_BIN_DIR = $(CI_CACHE_DIR)/bin

$(_BIN_DIR)/docker-compose:
	mkdir -p $(_BIN_DIR)
	curl -L https://github.com/docker/compose/releases/download/$(DOCKER_COMPOSE_VERSION)/docker-compose-$(shell uname -s)-$(shell uname -m) > $(_BIN_DIR)/docker-compose
	chmod +x $(_BIN_DIR)/docker-compose

##############################################################################

_GENERATE_DEPS_DIR = $(shell pwd)/.tmp
$(_GENERATE_DEPS_DIR):
	mkdir $(_GENERATE_DEPS_DIR)

# Full paths to executables needed for 'make generate'
_GENERATE_DEPS_EXECUTABLES = $(_GENERATE_DEPS_DIR)/thriftrw-plugin-yarpc

# Special-case for local executables
$(_GENERATE_DEPS_DIR)/thriftrw-plugin-yarpc: ./encoding/thrift/thriftrw-plugin-yarpc/*.go $(_GENERATE_DEPS_DIR)
	go build -o $(_GENERATE_DEPS_DIR)/thriftrw-plugin-yarpc ./encoding/thrift/thriftrw-plugin-yarpc

define generatedeprule
_GENERATE_DEPS_EXECUTABLES += $(_GENERATE_DEPS_DIR)/$(shell basename $1)

$(_GENERATE_DEPS_DIR)/$(shell basename $1): vendor/$1/*.go glide.lock $(_GENERATE_DEPS_DIR)
	./scripts/vendor-build.sh $(_GENERATE_DEPS_DIR) $1
endef

$(foreach i,$(GENERATE_DEPENDENCIES),$(eval $(call generatedeprule,$(i))))

THRIFTRW = $(_GENERATE_DEPS_DIR)/thriftrw

##############################################################################

.PHONY: build
build:
	go build $(PACKAGES)

.PHONY: generate
generate: $(_GENERATE_DEPS_EXECUTABLES)
	PATH=$(_GENERATE_DEPS_DIR):$$PATH ./scripts/generate.sh

.PHONY: nogogenerate
nogogenerate:
	$(eval NOGOGENERATE_LOG := $(shell mktemp -t nogogenerate.XXXXX))
	@grep -n \/\/go:generate $(GO_FILES) 2>&1 > $(NOGOGENERATE_LOG) || true
	@[ ! -s "$(NOGOGENERATE_LOG)" ] || (echo "do not use //go:generate, add to scripts/generate.sh instead:" | cat - $(NOGOGENERATE_LOG) && false)

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

.PHONY: verify_version
verify_version:
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
lint:
ifdef SHOULD_LINT
	@$(MAKE) nogogenerate gofmt govet golint staticcheck errcheck verify_version
else
	@echo "Linting not enabled on go $(GO_VERSION)"
endif

.PHONY: lintbins
lintbins:
ifdef SHOULD_LINT
	@go get github.com/golang/lint/golint
	@go get honnef.co/go/tools/cmd/staticcheck
	@go get github.com/kisielk/errcheck
endif

.PHONY: coverbins
coverbins:
	@go get github.com/wadey/gocovmerge
	@go get github.com/mattn/goveralls
	@go get golang.org/x/tools/cmd/cover

.PHONY: install
install:
	# all we want is go get -u github.com/Masterminds/glide
	# but have to pin to 0.12.3 due to https://github.com/Masterminds/glide/issues/745
	./scripts/glide-install.sh
	glide install


.PHONY: test
test: $(THRIFTRW)
	PATH=$(_GENERATE_DEPS_DIR):$$PATH go test -race $(PACKAGES)


.PHONY: cover
cover: $(THRIFTRW)
	PATH=$(_GENERATE_DEPS_DIR):$$PATH ./scripts/cover.sh $(shell go list $(PACKAGES))
	go tool cover -html=cover.out -o cover.html

.PHONY: goveralls
goveralls:
	goveralls -coverprofile=cover.out -service=travis-ci

.PHONY: examples
examples:
	$(MAKE) -C internal/examples

.PHONY: crossdock
crossdock: $(_BIN_DIR)/docker-compose
	PATH=$(_BIN_DIR):$$PATH docker-compose kill go
	PATH=$(_BIN_DIR):$$PATH docker-compose rm -f go
	PATH=$(_BIN_DIR):$$PATH docker-compose build go
	PATH=$(_BIN_DIR):$$PATH docker-compose run crossdock


.PHONY: crossdock-fresh
crossdock-fresh: $(_BIN_DIR)/docker-compose
	PATH=$(_BIN_DIR):$$PATH docker-compose kill
	PATH=$(_BIN_DIR):$$PATH docker-compose rm --force
	PATH=$(_BIN_DIR):$$PATH docker-compose pull
	PATH=$(_BIN_DIR):$$PATH docker-compose build
	PATH=$(_BIN_DIR):$$PATH docker-compose run crossdock

.PHONY: crossdock-logs
crossdock-logs:
	PATH=$(_BIN_DIR):$$PATH docker-compose logs

.PHONY: docker-build
docker-build:
	docker build -t yarpc_go .

.PHONY: docker-test
docker-test: docker-build
	docker run yarpc_go make test

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
	docker tag "$(CI_DOCKER_IMAGE)" "$(DOCKER_IMAGE)"
	docker login -e "$(DOCKER_EMAIL)" -u "$(DOCKER_USER)" -p "$(DOCKER_PASS)"
	docker push "$(DOCKER_IMAGE)"
endif

.PHONY: ci-install
ci-install: install
ifdef CI_LINT
	@$(MAKE) lintbins
endif
ifdef CI_COVER
	@$(MAKE) coverbins
endif

.PHONY: ci-run
ci-run:
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
