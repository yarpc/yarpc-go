OUTBOUND ?= tchannel

.PHONY: all
all: run

.PHONY: run
run:
	cat test.txt | go run main.go -outbound $(OUTBOUND)

