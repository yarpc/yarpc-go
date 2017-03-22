#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

protoc_with_imports() {
  protoc \
    -I vendor/github.com/gogo/protobuf \
    -I vendor/github.com/gogo/protobuf/protobuf \
    -I . \
    --${1}_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor,Mgogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto:. \
  ${@:2}
}

protoc_with_imports gogoslick encoding/x/protobuf/internal/wirepb/wire.proto
protoc_with_imports gogoslick internal/examples/protobuf/examplepb/example.proto
protoc_with_imports yarpc-go internal/examples/protobuf/examplepb/example.proto

update-license encoding/x/protobuf/internal/wirepb/wire.pb.go
update-license internal/examples/protobuf/examplepb/example.pb.go
update-license internal/examples/protobuf/examplepb/example.pb.yarpc.go

touch encoding/x/protobuf/internal/wirepb/.nocover
