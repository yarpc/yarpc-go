#!/bin/bash

set -Ee

error() {
  echo "error: $@" >&2
  exit 1
}

check_which() {
  if ! which $@ >/dev/null; then
    error "$@ not installed"
  fi
}

check_file_exists() {
  if [ ! -f "$1" ] ; then
    error "$1 does not exist"
  fi
}

check_which protoc
check_which protoc-gen-yarpc-go

google_protobuf_include_path="$(cd "$(dirname "$(which protoc)")/../include" && pwd)"
check_file_exists "${google_protobuf_include_path}/google/protobuf/descriptor.proto"
check_file_exists "${GOPATH}/src/go.uber.org/yarpc/yarpc.proto"

protoc \
  -I . \
  -I "${google_protobuf_include_path}" \
  -I "${GOPATH}/src" \
  --yarpc-go_out=Mgoogle/protobuf/descriptor.proto=github.com/golang/protobuf/protoc-gen-go/descriptor:. \
  $@
