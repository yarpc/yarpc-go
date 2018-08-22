ifdef RUN
SERVICE_TEST_FLAGS += --no-validate-output
endif
ifdef V
SERVICE_TEST_FLAGS += --debug
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
	streaming

.PHONY: install
install:
	go install ../service-test

.PHONY: build-json-keyvalue
build-json-keyvalue:
	go build -o json-keyvalue/client/client json-keyvalue/client/main.go
	go build -o json-keyvalue/server/server json-keyvalue/server/main.go

.PHONY: json-keyvalue-http
json-keyvalue-http: install build-json-keyvalue
	TRANSPORT=http $(SERVICE_TEST) --dir json-keyvalue

.PHONY: json-keyvalue-tchannel
json-keyvalue-tchannel: install build-json-keyvalue
	TRANSPORT=tchannel $(SERVICE_TEST) --dir json-keyvalue

.PHONY: json-keyvalue-grpc
json-keyvalue-grpc: install build-json-keyvalue
	TRANSPORT=grpc $(SERVICE_TEST) --dir json-keyvalue

.PHONY: protobuf-build
build-protobuf:
	go build -o protobuf/protobuf protobuf/main.go

.PHONY: protobuf
protobuf-http: install build-protobuf
	TRANSPORT=http $(SERVICE_TEST) --dir protobuf

.PHONY: protobuf
protobuf-tchannel: install build-protobuf
	TRANSPORT=tchannel $(SERVICE_TEST) --dir protobuf

.PHONY: protobuf
protobuf-grpc: install build-protobuf
	TRANSPORT=grpc $(SERVICE_TEST) --dir protobuf

.PHONY: protobuf
protobuf-google-grpc: install build-protobuf
	TRANSPORT=grpc GOOGLE_GRPC=--google-grpc $(SERVICE_TEST) --dir protobuf

.PHONY: thrift-hello
thrift-hello: install
	go build -o thrift-hello/hello/hello thrift-hello/hello/main.go
	$(SERVICE_TEST) --dir thrift-hello

.PHONY: build-thrift-keyvalue
build-thrift-keyvalue:
	go build -o thrift-keyvalue/keyvalue/client/client thrift-keyvalue/keyvalue/client/cache.go thrift-keyvalue/keyvalue/client/main.go
	go build -o thrift-keyvalue/keyvalue/server/server thrift-keyvalue/keyvalue/server/main.go

.PHONY: thrift-keyvalue-http
thrift-keyvalue-http: install build-thrift-keyvalue
	TRANSPORT=http $(SERVICE_TEST) --dir thrift-keyvalue

.PHONY: thrift-keyvalue-tchannel
thrift-keyvalue-tchannel: install build-thrift-keyvalue
	TRANSPORT=tchannel $(SERVICE_TEST) --dir thrift-keyvalue

.PHONY: thrift-keyvalue-grpc
thrift-keyvalue-grpc: install build-thrift-keyvalue
	TRANSPORT=grpc $(SERVICE_TEST) --dir thrift-keyvalue

.PHONY: build-streaming
build-streaming:
	go build -o streaming/client/client streaming/client/main.go
	go build -o streaming/server/server streaming/server/main.go

.PHONY: streaming
streaming: install build-streaming
	$(SERVICE_TEST) --dir streaming
