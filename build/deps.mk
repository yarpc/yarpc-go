# List of vendored go executables needed for make generate
GEN_GO_BIN_DEPS = \
	github.com/golang/mock/mockgen \
	github.com/gogo/protobuf/protoc-gen-gogoslick \
	github.com/uber/tchannel-go/thrift/thrift-gen \
	golang.org/x/tools/cmd/stringer \
	go.uber.org/thriftrw \
	go.uber.org/tools/update-license

# List of vendored go executables needed for other operations
EXTRA_GO_BIN_DEPS = \
	github.com/kisielk/errcheck \
	github.com/golang/lint/golint \
	github.com/mattn/goveralls \
	github.com/wadey/gocovmerge \
	golang.org/x/tools/cmd/cover \
	honnef.co/go/tools/cmd/staticcheck

# all we want is go get -u github.com/Masterminds/glide
# but have to pin to 0.12.3 due to https://github.com/Masterminds/glide/issues/745
GLIDE_VERSION := 0.12.3
THRIFT_VERSION := 1.0.0-dev
PROTOC_VERSION := 3.2.0

GLIDE_OS := $(UNAME_OS)
THRIFT_OS := $(UNAME_OS)
PROTOC_OS := $(UNAME_OS)

GLIDE_ARCH := $(UNAME_ARCH)
THRIFT_ARCH := $(UNAME_ARCH)
PROTOC_ARCH := $(UNAME_ARCH)

ifeq ($(UNAME_OS),Darwin)
GLIDE_OS := darwin
PROTOC_OS := osx
else
GLIDE_OS = linux
PROTOC_OS = linux
endif

ifeq ($(UNAME_ARCH),x86_64)
GLIDE_ARCH = amd64
endif

GLIDE = $(BIN)/glide
THRIFT = $(BIN)/thrift
PROTOC = $(BIN)/protoc
GEN_BINS = $(THRIFT) $(PROTOC)
EXTRA_BINS = $(GLIDE)

$(GLIDE):
	@mkdir -p $(BIN)
	@mkdir -p $(TMP_DIR)/glide
	curl -L https://github.com/Masterminds/glide/releases/download/v$(GLIDE_VERSION)/glide-v$(GLIDE_VERSION)-$(GLIDE_OS)-$(GLIDE_ARCH).tar.gz > $(TMP_DIR)/glide/glide.tar.gz
	cd $(TMP_DIR)/glide; tar xzf glide.tar.gz
	mv $(TMP_DIR)/glide/$(GLIDE_OS)-$(GLIDE_ARCH)/glide $(GLIDE)
	@rm -rf $(TMP_DIR)/glide

$(THRIFT):
	@mkdir -p $(BIN)
	@mkdir -p $(TMP_DIR)/thrift
	curl -L "https://github.com/uber/tchannel-go/releases/download/thrift-v$(THRIFT_VERSION)/thrift-1-$(THRIFT_OS)-$(THRIFT_ARCH).tar.gz" > $(TMP_DIR)/thrift/thrift.tar.gz
	tar -C $(TMP_DIR)/thrift -xzf $(TMP_DIR)/thrift/thrift.tar.gz
	mv $(TMP_DIR)/thrift/thrift-1 $(THRIFT)
	@rm -rf $(TMP_DIR)/thrift

$(PROTOC):
	@mkdir -p $(BIN)
	@mkdir -p $(TMP_DIR)/protoc
	curl -L "https://github.com/google/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-$(PROTOC_ARCH).zip" > $(TMP_DIR)/protoc/protoc.zip
	cd $(TMP_DIR)/protoc; unzip $(TMP_DIR)/protoc/protoc.zip
	mv $(TMP_DIR)/protoc/bin/protoc $(PROTOC)
	@rm -rf $(TMP_DIR)/protoc

define generatedeprule
GEN_BINS += $(BIN)/$(shell basename $1)
endef

define extradeprule
EXTRA_BINS += $(BIN)/$(shell basename $1)
endef

define deprule
$(BIN)/$(shell basename $1): glide.lock $(GLIDE)
	@mkdir -p $(BIN)
	PATH=$(BIN):$(PATH) ./scripts/vendor-build.sh $(BIN) $1
endef

$(foreach i,$(GEN_GO_BIN_DEPS),$(eval $(call generatedeprule,$(i))))
$(foreach i,$(GEN_GO_BIN_DEPS),$(eval $(call deprule,$(i))))
$(foreach i,$(EXTRA_GO_BIN_DEPS),$(eval $(call extradeprule,$(i))))
$(foreach i,$(EXTRA_GO_BIN_DEPS),$(eval $(call deprule,$(i))))

THRIFTRW = $(BIN)/thriftrw
GOLINT = $(BIN)/golint
ERRCHECK = $(BIN)/errcheck
STATICCHECK = $(BIN)/staticcheck
COVER = $(BIN)/cover
GOCOVMERGE = $(BIN)/gocovmerge
GOVERALLS = $(BIN)/goveralls

##############################################################################

.PHONY: deps
deps: __predeps $(GEN_BINS) $(EXTRA_BINS)

.PHONY: __predeps
__predeps: $(GLIDE)
	PATH=$$PATH:$(BIN) glide install
