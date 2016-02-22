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


.PHONY: crossdock
crossdock:
	docker-compose kill yarpc-go
	docker-compose rm -f yarpc-go
	docker-compose build yarpc-go
	docker-compose run crossdock


.PHONY: crossdock-fresh
crossdock-fresh: install
	docker-compose kill
	docker-compose rm --force
	docker-compose pull
	docker-compose build
	docker-compose run crossdock


.PHONY: install_ci
install_ci: install
	go get github.com/axw/gocov/gocov
	go get github.com/mattn/goveralls
	go get golang.org/x/tools/cmd/cover


.PHONY: test_ci
test_ci:
	goveralls -service=travis-ci -v $(PACKAGES)
