# Paths besides auto-detected generated files that should be excluded from
# lint results.
LINT_EXCLUDES_EXTRAS =

# Regex for 'go vet' rules to ignore
GOVET_IGNORE_RULES = \
	possible formatting directive in Error call

# List of executables needed for 'make generate'
GENERATE_DEPENDENCIES = \
	github.com/golang/mock/mockgen \
	github.com/golang/protobuf/protoc-gen-go \
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

##############################################################################

_GENERATE_DEPS_DIR = $(shell pwd)/.tmp
$(_GENERATE_DEPS_DIR):
	mkdir $(_GENERATE_DEPS_DIR)

# Full paths to executables needed for 'make generate'
_GENERATE_DEPS_EXECUTABLES = $(_GENERATE_DEPS_DIR)/thriftrw-plugin-yarpc $(_GENERATE_DEPS_DIR)/protoc-gen-yarpc-go

# Special-case for local executables
$(_GENERATE_DEPS_DIR)/thriftrw-plugin-yarpc: ./encoding/thrift/thriftrw-plugin-yarpc/*.go $(_GENERATE_DEPS_DIR)
	go build -o $(_GENERATE_DEPS_DIR)/thriftrw-plugin-yarpc ./encoding/thrift/thriftrw-plugin-yarpc

$(_GENERATE_DEPS_DIR)/protoc-gen-yarpc-go: ./encoding/x/protobuf/protoc-gen-yarpc-go/*.go $(_GENERATE_DEPS_DIR)
	go build -o $(_GENERATE_DEPS_DIR)/protoc-gen-yarpc-go ./encoding/x/protobuf/protoc-gen-yarpc-go

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

.PHONY: protogenerate
protogenerate: $(_GENERATE_DEPS_EXECUTABLES)
	@echo "TODO: merge with make generate once apache thrift issues fixed"
	@command -v protoc >/dev/null || (echo "protoc must be installed" && false)
	@protoc --version | grep 'libprotoc 3\.' >/dev/null || (echo "protoc must be version 3" && false)
	PATH=$(_GENERATE_DEPS_DIR):$$PATH ./scripts/protogenerate.sh

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
	@go get github.com/golang/lint/golint
	$(eval LINT_LOG := $(shell mktemp -t golint.XXXXX))
	@cat /dev/null > $(LINT_LOG)
	@$(foreach pkg, $(PACKAGES), golint $(pkg) | $(FILTER_LINT) >> $(LINT_LOG) || true;)
	@[ ! -s "$(LINT_LOG)" ] || (echo "golint failed:" | cat - $(LINT_LOG) && false)

.PHONY: staticcheck
staticcheck:
	@go get honnef.co/go/tools/cmd/staticcheck
	$(eval STATICCHECK_LOG := $(shell mktemp -t staticcheck.XXXXX))
	@staticcheck $(PACKAGES) 2>&1 | $(FILTER_LINT) > $(STATICCHECK_LOG) || true
	@[ ! -s "$(STATICCHECK_LOG)" ] || (echo "staticcheck failed:" | cat - $(STATICCHECK_LOG) && false)

.PHONY: errcheck
errcheck:
	@go get github.com/kisielk/errcheck
	$(eval ERRCHECK_LOG := $(shell mktemp -t errcheck.XXXXX))
	@errcheck $(ERRCHECK_FLAGS) $(PACKAGES) 2>&1 | $(FILTER_LINT) | $(FILTER_ERRCHECK) > $(ERRCHECK_LOG) || true
	@[ ! -s "$(ERRCHECK_LOG)" ] || (echo "errcheck failed:" | cat - $(ERRCHECK_LOG) && false)

.PHONY: lint
lint: nogogenerate gofmt govet golint staticcheck errcheck

.PHONY: install
install:
	# all we want is go get -u github.com/Masterminds/glide
	# but have to pin to 0.12.3 due to https://github.com/Masterminds/glide/issues/745
	./scripts/glide-install.sh
	glide install


.PHONY: prototest
prototest:
	$(MAKE) -C internal/examples/protobuf-keyvalue test

.PHONY: test
test: verify_version $(THRIFTRW) prototest
	PATH=$(_GENERATE_DEPS_DIR):$$PATH go test -race $(PACKAGES)


.PHONY: cover
cover:
	./scripts/cover.sh $(shell go list $(PACKAGES))
	go tool cover -html=cover.out -o cover.html


# This is not part of the regular test target because we don't want to slow it
# down.
.PHONY: test-examples
test-examples: build
	make -C internal/examples


.PHONY: crossdock
crossdock:
	docker-compose kill go
	docker-compose rm -f go
	docker-compose build go
	docker-compose run crossdock


.PHONY: crossdock-fresh
crossdock-fresh: install
	docker-compose kill
	docker-compose rm --force
	docker-compose pull
	docker-compose build
	docker-compose run crossdock

.PHONY: docker-build
docker-build:
	docker build -t yarpc_go .

.PHONY: docker-test
docker-test: docker-build
	docker run yarpc_go make test

.PHONY: test_ci
test_ci: install verify_version $(THRIFTRW)
	PATH=$(_GENERATE_DEPS_DIR):$$PATH ./scripts/cover.sh $(shell go list $(PACKAGES))

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
