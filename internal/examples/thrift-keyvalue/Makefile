TRANSPORT ?= http

.PHONY: all
all: test

.PHONY: run
run:
	TRANSPORT=$(TRANSPORT) service-test --no-verify-output

.PHONY:
test:
	TRANSPORT=$(TRANSPORT) service-test
