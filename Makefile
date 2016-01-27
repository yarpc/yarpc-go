project := yarpc-go


.PHONY: install
install:
	go build ./...


.PHONY: test
test:
	go test ./...


.PHONY: xlang
xlang:
	docker-compose run xlang
