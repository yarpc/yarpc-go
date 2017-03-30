EXAMPLES = thrift-hello thrift-keyvalue json-keyvalue thrift-oneway
TRANSPORTS = http tchannel grpc

ENV = env
CRAM = $(ENV)/bin/cram

.PHONY: all
all: | build test

.PHONY: test
test: $(CRAM)
	@$(foreach transport, $(TRANSPORTS), \
		echo "---- Running tests over $(transport) ----" && \
		TRANSPORT=$(transport) $(CRAM) . &&) echo "---- Done ----"
	@make -C protobuf test

.PHONY: build
build:
	@$(foreach example, $(EXAMPLES), \
		echo "---- Building $(example) ----"; \
		make -C $(example);)

$(CRAM): $(ENV)/bin/activate
	. $(ENV)/bin/activate && pip install cram

$(ENV)/bin/activate:
	virtualenv $(ENV)
