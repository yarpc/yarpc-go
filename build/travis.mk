DOCKER_CACHE_DIR := $(CACHE_DIR)/docker
CROSSDOCK_DOCKER_CACHE_FILE := $(DOCKER_CACHE_DIR)/$(CROSSDOCK_DOCKER_IMAGE)
DOCKER_CACHE_FILE := $(DOCKER_CACHE_DIR)/$(DOCKER_IMAGE)

.PHONY: travis-docker-load
travis-docker-load:
ifndef SUPPRESS_CROSSDOCK
	if [ -f $(CROSSDOCK_DOCKER_CACHE_FILE) ]; then gunzip -c $(CROSSDOCK_DOCKER_CACHE_FILE) | docker load; fi
endif
	if [ -f $(DOCKER_CACHE_FILE) ]; then gunzip -c $(DOCKER_CACHE_FILE) | docker load; fi

.PHONY: travis-docker-save
travis-docker-save:
	mkdir -p $(DOCKER_CACHE_DIR)
ifndef SUPPRESS_CROSSDOCK
	docker save $(shell docker history -q $(CROSSDOCK_DOCKER_IMAGE) | grep -v '<missing>') | gzip > $(CROSSDOCK_DOCKER_CACHE_FILE)
endif
	docker save $(shell docker history -q $(DOCKER_IMAGE) | grep -v '<missing>') | gzip > $(DOCKER_CACHE_FILE)

.PHONY: travis-docker-push
travis-docker-push:
ifndef SUPPRESS_CROSSDOCK
	./scripts/travis-docker-push.sh
endif
