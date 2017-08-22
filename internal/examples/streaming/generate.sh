#!/bin/bash

set -euo pipefail
#set -x

DIR="$(cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

if echo "${GOPATH}" | grep : >/dev/null; then
  echo "error: GOPATH can only contain one directory but is ${GOPATH}" >&2
  exit 1
fi

# Run protoc
#
# $1: plugin
# $2: file
# $3: other options
protoc_with_imports() {
  protoc \
    -I "${GOPATH}/src" \
    -I vendor \
    -I vendor/github.com/gogo/protobuf/protobuf \
    -I . \
    "--${1}_out=${3}Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor,Mgogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto:." \
  "${2}"
}

protoc_go() {
  protoc_with_imports "go" "${1}" ""
}

protoc_go_grpc() {
  protoc_with_imports "go" "${1}" "plugins=grpc,"
}

protoc_yarpc_go() {
  protoc_with_imports "yarpc-go" "${1}" ""
}

protoc_all() {
  protoc_go_grpc "${1}"
  protoc_yarpc_go "${1}"
}

go install ./encoding/protobuf/protoc-gen-yarpc-go

rm -f internal/examples/streaming/stream.pb.go
rm -f internal/examples/streaming/stream.pb.yarpc.go

protoc_all internal/examples/streaming/stream.proto
#vim eee/eee.pb.yarpc.go eee/eee.pb.go