project := yarpc-go

export GO15VENDOREXPERIMENT=1


.PHONY: build
build:
	go build `glide novendor`

.PHONY: install
install:
	glide --version || go get github.com/Masterminds/glide
	glide install
	go build `glide novendor`


.PHONY: test
test:
	go test `glide novendor`


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
