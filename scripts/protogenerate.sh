#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

protoc --go_out=. encoding/x/protobuf/internal/internal.proto
protoc --go_out=. internal/examples/protobuf-keyvalue/kv/kv.proto
protoc --yarpc-go_out=. internal/examples/protobuf-keyvalue/kv/kv.proto

update-license encoding/x/protobuf/internal/internal.pb.go
update-license internal/examples/protobuf-keyvalue/kv/kv.pb.go
update-license internal/examples/protobuf-keyvalue/kv/kv.pb.yarpc.go
