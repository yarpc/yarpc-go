project := yarpc-go


.PHONY: install
install:
	glide --version || go get github.com/Masterminds/glide
	GO15VENDOREXPERIMENT=1 glide install
	GO15VENDOREXPERIMENT=1 go build `GO15VENDOREXPERIMENT=1 glide novendor`


.PHONY: test
test:
	GO15VENDOREXPERIMENT=1 go test `GO15VENDOREXPERIMENT=1 glide novendor`


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
