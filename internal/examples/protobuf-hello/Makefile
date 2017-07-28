.PHONY: install
install:
	go get github.com/gogo/protobuf/protoc-gen-gogoslick
	go get go.uber.org/yarpc/encoding/protobuf/protoc-gen-yarpc-go

.PHONY: generate
generate:
	protoc --gogoslick_out=. --yarpc-go_out=. hello.proto
