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
	github.com/wadey/gocovmerge \
	golang.org/x/tools/cmd/cover \
	honnef.co/go/tools/cmd/staticcheck

# all we want is go get -u github.com/Masterminds/glide
# but have to pin to 0.12.3 due to https://github.com/Masterminds/glide/issues/745
GLIDE_VERSION := 0.12.3
THRIFT_VERSION := 1.0.0-dev
PROTOC_VERSION := 3.3.0
RAGEL_VERSION := 6.9

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

GLIDE_LIB = $(LIB)/glide-$(GLIDE_VERSION)
GLIDE_TAR = $(GLIDE_LIB)/glide.tar.gz
GLIDE = $(BIN)/glide
THRIFT_LIB = $(LIB)/thrift-$(THRIFT_VERSION)
THRIFT_TAR = $(THRIFT_LIB)/thrift.tar.gz
THRIFT = $(BIN)/thrift
PROTOC_LIB = $(LIB)/protoc-$(PROTOC_VERSION)
PROTOC_ZIP = $(PROTOC_LIB)/protoc.zip
PROTOC = $(BIN)/protoc
RAGEL_LIB = $(LIB)/ragel-$(RAGEL_VERSION)
RAGEL_TAR = $(RAGEL_LIB)/ragel.tar.gz
RAGEL = $(BIN)/ragel

GEN_BINS = $(THRIFT) $(PROTOC) $(RAGEL)
EXTRA_BINS = $(GLIDE)

$(RAGEL_TAR):
	@mkdir -p $(RAGEL_LIB)
	curl -L "https://www.colm.net/files/ragel/ragel-$(RAGEL_VERSION).tar.gz" > $(RAGEL_TAR)

$(RAGEL): $(RAGEL_TAR)
	@mkdir -p $(BIN)
	cd $(RAGEL_LIB); tar xzf $(RAGEL_TAR)
	cd $(RAGEL_LIB)/ragel-$(RAGEL_VERSION); ./configure --prefix=$(RAGEL_LIB) --disable-manual
	cd $(RAGEL_LIB)/ragel-$(RAGEL_VERSION); make install
	cp $(RAGEL_LIB)/bin/ragel $(RAGEL)

$(GLIDE_TAR):
	@mkdir -p $(GLIDE_LIB)
	curl -L "https://github.com/Masterminds/glide/releases/download/v$(GLIDE_VERSION)/glide-v$(GLIDE_VERSION)-$(GLIDE_OS)-$(GLIDE_ARCH).tar.gz" > $(GLIDE_TAR)

$(GLIDE): $(GLIDE_TAR)
	@mkdir -p $(BIN)
	cd $(GLIDE_LIB); tar xzf $(GLIDE_TAR)
	cp $(GLIDE_LIB)/$(GLIDE_OS)-$(GLIDE_ARCH)/glide $(GLIDE)

$(THRIFT_TAR):
	@mkdir -p $(THRIFT_LIB)
	curl -L "https://github.com/uber/tchannel-go/releases/download/thrift-v$(THRIFT_VERSION)/thrift-1-$(THRIFT_OS)-$(THRIFT_ARCH).tar.gz" > $(THRIFT_TAR)

$(THRIFT): $(THRIFT_TAR)
	@mkdir -p $(BIN)
	cd $(THRIFT_LIB); tar xzf $(THRIFT_TAR)
	cp $(THRIFT_LIB)/thrift-1 $(THRIFT)

$(PROTOC_ZIP):
	@mkdir -p $(PROTOC_LIB)
	curl -L "https://github.com/google/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-$(PROTOC_ARCH).zip" > $(PROTOC_ZIP)

$(PROTOC): $(PROTOC_ZIP)
	@mkdir -p $(BIN)
	cd $(PROTOC_LIB); unzip $(PROTOC_ZIP)
	cp $(PROTOC_LIB)/bin/protoc $(PROTOC)

define generatedeprule
GEN_BINS += $(BIN)/$(shell basename $1)
endef

define extradeprule
EXTRA_BINS += $(BIN)/$(shell basename $1)
endef

define deprule
ifdef SUPPRESS_DOCKER
$(BIN)/$(shell basename $1): glide.lock $(GLIDE)
	@mkdir -p $(BIN)
	PATH=$(BIN):$(PATH) ./scripts/vendor-build.sh $(BIN) $1
else
$(BIN)/$(shell basename $1): $(GLIDE)
	@mkdir -p $(BIN)
	PATH=$(BIN):$(PATH) ./scripts/vendor-build.sh $(BIN) $1
endif
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

.PHONY: predeps
predeps: $(GLIDE) $(THRIFT) $(PROTOC) $(RAGEL)

.PHONY: deps
deps: predeps glide $(GEN_BINS) $(EXTRA_BINS) ## install all dependencies

.PHONY: glide
glide: $(GLIDE) ## install glide dependencies
	PATH=$$PATH:$(BIN) glide install
