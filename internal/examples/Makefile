ifdef RUN
SERVICE_TEST := service-test --no-verify-output
else
SERVICE_TEST := service-test
endif

.PHONY: all
all: \
	json-keyvalue-http \
	json-keyvalue-tchannel \
	json-keyvalue-grpc \
	protobuf \
	thrift-hello \
	thrift-keyvalue-http \
	thrift-keyvalue-tchannel \
	thrift-keyvalue-grpc \
	thrift-oneway

.PHONY: install
install:
	go install ../service-test

.PHONY: json-keyvalue-http
json-keyvalue-http: install
	TRANSPORT=http $(SERVICE_TEST) --dir json-keyvalue

.PHONY: json-keyvalue-tchannel
json-keyvalue-tchannel: install
	TRANSPORT=tchannel $(SERVICE_TEST) --dir json-keyvalue

.PHONY: json-keyvalue-grpc
json-keyvalue-grpc: install
	TRANSPORT=grpc $(SERVICE_TEST) --dir json-keyvalue

.PHONY: protobuf
protobuf: install
	$(SERVICE_TEST) --dir protobuf

.PHONY: thrift-hello
thrift-hello: install
	$(SERVICE_TEST) --dir thrift-hello

.PHONY: thrift-keyvalue-http
thrift-keyvalue-http: install
	TRANSPORT=http $(SERVICE_TEST) --dir thrift-keyvalue

.PHONY: thrift-keyvalue-tchannel
thrift-keyvalue-tchannel: install
	TRANSPORT=tchannel $(SERVICE_TEST) --dir thrift-keyvalue

.PHONY: thrift-keyvalue-grpc
thrift-keyvalue-grpc: install
	TRANSPORT=grpc $(SERVICE_TEST) --dir thrift-keyvalue

.PHONY: thrift-oneway
thrift-oneway:
	GOBIN=/tmp go install thrift-oneway/main.go
