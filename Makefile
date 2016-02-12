PACKAGES := $(shell glide novendor)

export GO15VENDOREXPERIMENT=1


.PHONY: build
build:
	go build $(PACKAGES)

.PHONY: install
install:
	glide --version || go get github.com/Masterminds/glide
	glide install


.PHONY: test
test:
	go test $(PACKAGES)


.PHONY: xlang
xlang:
	docker-compose run xlang


.PHONY: xlang-fresh
xlang-fresh:
	docker-compose kill
	docker-compose rm --force
	docker-compose pull
	docker-compose build
	docker-compose run xlang

##############################################################################
# CI

.PHONY: install_ci
install_ci: install
	go get github.com/axw/gocov/gocov
	go get github.com/mattn/goveralls
	go get golang.org/x/tools/cmd/cover

# Tests don't need to be run separately because goveralls takes care of
# running them.

.PHONY: test_ci
test_ci:
	goveralls -service=travis-ci -v $(PACKAGES)
