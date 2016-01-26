project := yarpc-go


.PHONY: install
install:
	go build ./...


.PHONY: test
test:
	go test ./...
