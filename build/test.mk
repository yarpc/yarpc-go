# Paths besides auto-detected generated files that should be excluded from
# lint results.
LINT_EXCLUDES_EXTRAS =

# Regex for 'go vet' rules to ignore
GOVET_IGNORE_RULES = \
	possible formatting directive in Error call

ERRCHECK_FLAGS ?= -ignoretests
ERRCHECK_EXCLUDES := \.Close\(\) \.Stop\(\)
FILTER_ERRCHECK := grep -v $(patsubst %,-e %, $(ERRCHECK_EXCLUDES))

GEN_BINS_INTERNAL = $(BIN)/thriftrw-plugin-yarpc $(BIN)/protoc-gen-yarpc-go

$(BIN)/thriftrw-plugin-yarpc: ./encoding/thrift/thriftrw-plugin-yarpc/*.go
	@mkdir -p $(BIN)
	go build -o $(BIN)/thriftrw-plugin-yarpc ./encoding/thrift/thriftrw-plugin-yarpc

$(BIN)/protoc-gen-yarpc-go: ./encoding/x/protobuf/protoc-gen-yarpc-go/*.go
	@mkdir -p $(BIN)
	go build -o $(BIN)/protoc-gen-yarpc-go ./encoding/x/protobuf/protoc-gen-yarpc-go

.PHONY: build
build: __eval_packages
	go build $(PACKAGES)

.PHONY: generate
generate: $(GEN_BINS) $(GEN_BINS_INTERNAL)
	@go get github.com/golang/mock/mockgen
	@PATH=$(BIN):$$PATH ./scripts/generate.sh

.PHONY: nogogenerate
nogogenerate: __eval_go_files
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
gofmt: __eval_go_files
	$(eval FMT_LOG := $(shell mktemp -t gofmt.XXXXX))
	@PATH=$(BIN):$$PATH gofmt -e -s -l $(GO_FILES) | $(FILTER_LINT) > $(FMT_LOG) || true
	@[ ! -s "$(FMT_LOG)" ] || (echo "gofmt failed:" | cat - $(FMT_LOG) && false)

.PHONY: govet
govet: __eval_packages __eval_go_files
	$(eval VET_LOG := $(shell mktemp -t govet.XXXXX))
	@go vet $(PACKAGES) 2>&1 \
		| grep -v '^exit status' \
		| grep -v "$(GOVET_IGNORE_RULES)" \
		| $(FILTER_LINT) > $(VET_LOG) || true
	@[ ! -s "$(VET_LOG)" ] || (echo "govet failed:" | cat - $(VET_LOG) && false)

.PHONY: golint
golint: $(GOLINT) __eval_packages __eval_go_files
	$(eval LINT_LOG := $(shell mktemp -t golint.XXXXX))
	@for pkg in $(PACKAGES); do \
		PATH=$(BIN):$$PATH golint $$pkg | $(FILTER_LINT) >> $(LINT_LOG) || true; \
	done
	@[ ! -s "$(LINT_LOG)" ] || (echo "golint failed:" | cat - $(LINT_LOG) && false)

.PHONY: staticcheck
staticcheck: $(STATICCHECK) __eval_packages __eval_go_files
	$(eval STATICCHECK_LOG := $(shell mktemp -t staticcheck.XXXXX))
	@PATH=$(BIN):$$PATH staticcheck $(PACKAGES) 2>&1 | $(FILTER_LINT) > $(STATICCHECK_LOG) || true
	@[ ! -s "$(STATICCHECK_LOG)" ] || (echo "staticcheck failed:" | cat - $(STATICCHECK_LOG) && false)

.PHONY: errcheck
errcheck: $(ERRCHECK) __eval_packages __eval_go_files
	$(eval ERRCHECK_LOG := $(shell mktemp -t errcheck.XXXXX))
	@PATH=$(BIN):$$PATH errcheck $(ERRCHECK_FLAGS) $(PACKAGES) 2>&1 | $(FILTER_LINT) | $(FILTER_ERRCHECK) > $(ERRCHECK_LOG) || true
	@[ ! -s "$(ERRCHECK_LOG)" ] || (echo "errcheck failed:" | cat - $(ERRCHECK_LOG) && false)

.PHONY: verifyversion
verifyversion:
	$(eval CHANGELOG_VERSION := $(shell grep '^v[0-9]' CHANGELOG.md | head -n1 | cut -d' ' -f1))
	$(eval INTHECODE_VERSION := $(shell perl -ne '/^const Version.*"([^"]+)".*$$/ && print "v$$1\n"' version.go))
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
lint: generatenodiff nogogenerate gofmt govet golint staticcheck errcheck verifyversion

.PHONY: test
test: $(THRIFTRW) __eval_packages
	PATH=$(BIN):$$PATH go test -race $(PACKAGES)


.PHONY: cover
cover: $(THRIFTRW) $(GOCOVMERGE) $(COVER) __eval_packages
	PATH=$(BIN):$$PATH ./scripts/cover.sh $(PACKAGES)
	go tool cover -html=cover.out -o cover.html

.PHONY: goveralls
goveralls: cover $(GOVERALLS)
	PATH=$(BIN):$$PATH goveralls -coverprofile=cover.out -service=travis-ci

.PHONY: examples
examples:
	RUN=$(RUN) $(MAKE) -C internal/examples protobuf

.PHONY: __eval_packages
__eval_packages:
	$(eval PACKAGES := $(shell go list ./... | grep -v go\.uber\.org\/yarpc\/vendor))

.PHONY: __eval_go_files
__eval_go_files:
	$(eval GO_FILES := $(shell find . -name '*.go' | sed 's/^\.\///' | grep -v -e ^vendor\/ -e ^\.glide\/))
	$(eval GENERATED_GO_FILES := $(shell \
		find $(GO_FILES) \
		-exec sh -c 'head -n30 {} | grep "Code generated by\|Autogenerated by\|Automatically generated by\|@generated" >/dev/null' \; \
		-print))
	$(eval FILTER_LINT := grep -v $(patsubst %,-e %, $(GENERATED_GO_FILES) $(LINT_EXCLUDES_EXTRAS)))
