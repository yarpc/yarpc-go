.PHONY: generate
generate:
ifeq ($(shell which prototool),)
	@go get -u github.com/uber/prototool/internal/cmd/prototool
endif
	@prototool generate