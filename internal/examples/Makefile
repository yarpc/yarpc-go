ifdef RUN
SERVICE_TEST_FLAGS += --no-verify-output
endif
ifdef V
SERVICE_TEST_FLAGS += --verbose
endif
SERVICE_TEST := service-test $(SERVICE_TEST_FLAGS)

.PHONY: all
all: \
	json-keyvalue-http \
	json-keyvalue-tchannel \
	json-keyvalue-grpc \
	protobuf-http \
	protobuf-tchannel \
	protobuf-grpc \
	protobuf-google-grpc \
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
protobuf-http: install
	TRANSPORT=http $(SERVICE_TEST) --dir protobuf

.PHONY: protobuf
protobuf-tchannel: install
	TRANSPORT=tchannel $(SERVICE_TEST) --dir protobuf

.PHONY: protobuf
protobuf-grpc: install
	TRANSPORT=grpc $(SERVICE_TEST) --dir protobuf

.PHONY: protobuf
protobuf-google-grpc: install
	TRANSPORT=grpc GOOGLE_GRPC=--google-grpc $(SERVICE_TEST) --dir protobuf

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
