#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

if echo "${GOPATH}" | grep : >/dev/null; then
  echo "error: GOPATH can only contain one directory but is ${GOPATH}" >&2
  exit 1
fi

protoc_with_imports() {
  protoc \
    -I "${GOPATH}/src" \
    -I vendor \
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
