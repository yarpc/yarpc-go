DOCKER_COMPOSE_VERSION := 1.10.0
DOCKER_COMPOSE_OS := $(UNAME_OS)
DOCKER_COMPOSE_ARCH := $(UNAME_ARCH)

DOCKER_COMPOSE = $(BIN)/docker-compose

$(DOCKER_COMPOSE):
	@mkdir -p $(BIN)
	curl -L https://github.com/docker/compose/releases/download/$(DOCKER_COMPOSE_VERSION)/docker-compose-$(DOCKER_COMPOSE_OS)-$(DOCKER_COMPOSE_ARCH) > $(DOCKER_COMPOSE)
	@chmod +x $(DOCKER_COMPOSE)

.PHONY: crossdock
crossdock: $(DOCKER_COMPOSE) ## run crossdock
	$(DOCKER_COMPOSE) kill go
	$(DOCKER_COMPOSE) rm -f go
	$(DOCKER_COMPOSE) build go
	$(DOCKER_COMPOSE) run crossdock


.PHONY: crossdock-fresh
crossdock-fresh: $(DOCKER_COMPOSE) ## run crossdock from scratch
	$(DOCKER_COMPOSE) kill
	$(DOCKER_COMPOSE) rm --force
	$(DOCKER_COMPOSE) pull
	$(DOCKER_COMPOSE) build
	$(DOCKER_COMPOSE) run crossdock

.PHONY: crossdock-logs
crossdock-logs: $(DOCKER_COMPOSE) ## get crossdock logs
	$(DOCKER_COMPOSE) logs
