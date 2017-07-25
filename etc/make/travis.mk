CROSSDOCK_DOCKER_IMAGE := yarpc_go

DOCKER_RUN_FLAGS += -e TRAVIS_JOB_ID -e TRAVIS_PULL_REQUEST

.PHONY: travis-docker-push
travis-docker-push: ## push crossdock docker image from travis
ifndef SUPPRESS_CROSSDOCK
	PATH=$$PATH:$(BIN) ./etc/bin/travis-docker-push.sh "$(CROSSDOCK_DOCKER_IMAGE)"
endif
