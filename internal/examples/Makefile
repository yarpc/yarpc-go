.PHONY: all
all: test

.PHONY: test
test:
	go install ../service-test
	TRANSPORT=http make -C json-keyvalue
	TRANSPORT=tchannel make -C json-keyvalue
	make -C protobuf
	make -C thrift-hello
	TRANSPORT=http make -C thrift-keyvalue
	TRANSPORT=tchannel make -C thrift-keyvalue
	make -C thrift-oneway
