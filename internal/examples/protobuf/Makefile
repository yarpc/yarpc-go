.PHONY: all
all: test

.PHONY: run
run:
	cat test-input.txt | go run main.go

.PHONY:
test:
	$(eval TEST_ACTUAL_OUTPUT_TXT := $(shell mktemp -t protobuf-output.XXXXX))
	@cat test-input.txt | go run main.go > $(TEST_ACTUAL_OUTPUT_TXT)
	@diff test-expected-output.txt $(TEST_ACTUAL_OUTPUT_TXT) || echo "error: diff from expected output"
