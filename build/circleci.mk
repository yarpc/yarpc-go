DOCKER_VERSION := 17.03.1-ce
DOCKER_OS := $(UNAME_OS)
DOCKER_ARCH := $(UNAME_ARCH)

DOCKER_LIB = $(LIB)/docker-$(DOCKER_VERSION)
DOCKER_TAR = $(DOCKER_LIB)/docker.tar.gz
DOCKER = $(BIN)/docker

$(DOCKER_TAR):
	@mkdir -p $(DOCKER_LIB)
	curl -L "https://get.docker.com/builds/$(DOCKER_OS)/$(DOCKER_ARCH)/docker-$(DOCKER_VERSION).tgz" > $(DOCKER_TAR)

$(DOCKER): $(DOCKER_TAR)
	@mkdir -p $(BIN)
	cd $(DOCKER_LIB); tar xzf $(DOCKER_TAR)
	cp $(DOCKER_LIB)/docker/docker $(DOCKER)

.PHONY: circleci-deps
circleci-deps: $(DOCKER)
