project := yarpc-go

.PHONY: build
build:
	GO15VENDOREXPERIMENT=1 go build `glide novendor`

.PHONY: install
install:
	glide --version || go get github.com/Masterminds/glide
	GO15VENDOREXPERIMENT=1 glide install
	GO15VENDOREXPERIMENT=1 go build `glide novendor`


.PHONY: test
test:
	GO15VENDOREXPERIMENT=1 go test `glide novendor`


.PHONY: xlang
xlang:
	docker-compose run xlang
