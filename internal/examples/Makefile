ifdef RUN
SERVICE_TEST_FLAGS += --no-verify-output
endif
ifdef V
SERVICE_TEST_FLAGS += --debug
endif
SERVICE_TEST := service-test $(SERVICE_TEST_FLAGS)

.PHONY: all
all: json-keyvalue-http json-keyvalue-tchannel protobuf thrift-hello thrift-keyvalue-http thrift-keyvalue-tchannel thrift-oneway

.PHONY: install
install:
	go install ../service-test

json-keyvalue/client/client: json-keyvalue/client/main.go
	go build -o json-keyvalue/client/client json-keyvalue/client/main.go

json-keyvalue/server/server: json-keyvalue/server/main.go
	go build -o json-keyvalue/server/server json-keyvalue/server/main.go

.PHONY: json-keyvalue-http
json-keyvalue-http: install json-keyvalue/client/client json-keyvalue/server/server
	TRANSPORT=http $(SERVICE_TEST) --dir json-keyvalue

.PHONY: json-keyvalue-tchannel
json-keyvalue-tchannel: install json-keyvalue/client/client json-keyvalue/server/server
	TRANSPORT=tchannel $(SERVICE_TEST) --dir json-keyvalue

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

.PHONY: thrift-oneway
thrift-oneway:
	GOBIN=/tmp go install thrift-oneway/main.go
