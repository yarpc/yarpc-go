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
