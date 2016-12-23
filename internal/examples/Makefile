EXAMPLES = thrift-hello thrift-keyvalue json-keyvalue thrift-oneway
TRANSPORTS = http tchannel

ENV = env
CRAM = $(ENV)/bin/cram

.PHONY: all
all: | build test

.PHONY: test
test: $(CRAM)
	@$(foreach transport, $(TRANSPORTS), \
		echo "---- Running tests over $(transport) ----" && \
		TRANSPORT=$(transport) $(CRAM) . &&) echo "---- Done ----"

.PHONY: build
build:
	@$(foreach example, $(EXAMPLES), \
		echo "---- Building $(example) ----"; \
		make -C $(example);)

$(CRAM): $(ENV)
	. $(ENV)/bin/activate && pip install cram

$(ENV):
	virtualenv $(ENV)
