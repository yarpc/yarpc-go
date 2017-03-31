DOCKER_CACHE_DIR := $(CACHE_DIR)/docker
CROSSDOCK_DOCKER_CACHE_FILE := $(DOCKER_CACHE_DIR)/$(CROSSDOCK_DOCKER_IMAGE)
DOCKER_CACHE_FILE := $(DOCKER_CACHE_DIR)/$(DOCKER_IMAGE)

SERVICE_TEST_FLAGS += --no-kill
DOCKER_RUN_FLAGS += -e TRAVIS_JOB_ID -e TRAVIS_PULL_REQUEST

CROSSDOCK_DOCKER_IMAGE := yarpcgo_go

.PHONY: travis-docker-load
travis-docker-load: ## load docker images from travis cache
ifndef SUPPRESS_CROSSDOCK
	if [ -f $(CROSSDOCK_DOCKER_CACHE_FILE) ]; then gunzip -c $(CROSSDOCK_DOCKER_CACHE_FILE) | docker load; fi
endif
	if [ -f $(DOCKER_CACHE_FILE) ]; then gunzip -c $(DOCKER_CACHE_FILE) | docker load; fi

.PHONY: travis-docker-save
travis-docker-save: ## save docker images to travis cache
ifeq ($(TRAVIS_BRANCH),dev)
ifeq ($(TRAVIS_PULL_REQUEST),false)
	mkdir -p $(DOCKER_CACHE_DIR)
ifndef SUPPRESS_CROSSDOCK
	docker save $(shell docker history -q $(CROSSDOCK_DOCKER_IMAGE) | grep -v '<missing>') | gzip > $(CROSSDOCK_DOCKER_CACHE_FILE)
endif
	docker save $(shell docker history -q $(DOCKER_IMAGE) | grep -v '<missing>') | gzip > $(DOCKER_CACHE_FILE)
endif
endif

.PHONY: travis-docker-push
travis-docker-push: ## push crossdock docker image from travis
ifndef SUPPRESS_CROSSDOCK
	./scripts/travis-docker-push.sh "$(CROSSDOCK_DOCKER_IMAGE)"
endif
