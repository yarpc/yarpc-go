#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
cd "${DIR}"

if echo "${GOPATH}" | grep : >/dev/null; then
  echo "error: GOPATH can only contain one directory but is ${GOPATH}" >&2
  exit 1
fi

# Run stringer
#
# https://github.com/golang/go/issues/10249
#
# $1: type
# $2: go package
generate_stringer() {
  go install "${2}"
  stringer "-type=${1}" "${2}"
}

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
  protoc_with_imports "gogoslick" "${1}" ""
}

protoc_go_grpc() {
  protoc_with_imports "gogoslick" "${1}" "plugins=grpc,"
}

protoc_yarpc_go() {
  protoc_with_imports "yarpc-go" "${1}" ""
}

# Add "Generated by" header to Ragel-generated code.
#
# $1: path to the file
generated_by_ragel() {
	f=$(mktemp -t ragel.XXXXX)
	echo -e '// Code generated by ragel\n// @generated\n' | cat - "$1" > "$f"
	mv "$f" "$1"
}

# Strip thrift warnings.
strip_thrift_warnings() {
  grep -v '^\[WARNING:.*emphasize the signedness' | sed '/^\s*$/d'
}

mockgen -destination=api/middleware/middlewaretest/router.go -package=middlewaretest go.uber.org/yarpc/api/middleware Router,UnaryInbound,UnaryOutbound,OnewayInbound,OnewayOutbound
mockgen -destination=api/peer/peertest/list.go -package=peertest go.uber.org/yarpc/api/peer Chooser,List,ChooserList
mockgen -destination=api/peer/peertest/peer.go -package=peertest go.uber.org/yarpc/api/peer Identifier,Peer
mockgen -destination=api/peer/peertest/transport.go -package=peertest go.uber.org/yarpc/api/peer Transport,Subscriber
mockgen -destination=api/transport/transporttest/clientconfig.go -package=transporttest go.uber.org/yarpc/api/transport ClientConfig,ClientConfigProvider
mockgen -destination=api/transport/transporttest/handler.go -package=transporttest go.uber.org/yarpc/api/transport UnaryHandler,OnewayHandler
mockgen -destination=api/transport/transporttest/inbound.go -package=transporttest go.uber.org/yarpc/api/transport Inbound
mockgen -destination=api/transport/transporttest/outbound.go -package=transporttest go.uber.org/yarpc/api/transport UnaryOutbound,OnewayOutbound
mockgen -destination=api/transport/transporttest/router.go -package=transporttest go.uber.org/yarpc/api/transport Router,RouteTable
mockgen -destination=api/transport/transporttest/transport.go -package=transporttest go.uber.org/yarpc/api/transport Transport
mockgen -source=vendor/go.uber.org/thriftrw/protocol/protocol.go -destination=encoding/thrift/mock_protocol_test.go -package=thrift go.uber.org/thriftrw/protocol Protocol

generate_stringer ConnectionStatus ./api/peer
generate_stringer State ./pkg/lifecycle
generate_stringer Type ./api/transport

thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/echo.thrift
thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/oneway.thrift
thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/gauntlet.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-oneway internal/examples/thrift-oneway/sink.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-hello/hello internal/examples/thrift-hello/hello/echo.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-keyvalue/keyvalue internal/examples/thrift-keyvalue/keyvalue/kv.thrift
thriftrw --out=encoding/thrift encoding/thrift/internal.thrift
thriftrw --out=serialize serialize/internal.thrift

thriftrw --no-recurse --plugin=yarpc --out=encoding/thrift/thriftrw-plugin-yarpc/internal/tests encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common.thrift
thriftrw --no-recurse --plugin=yarpc --out=encoding/thrift/thriftrw-plugin-yarpc/internal/tests encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic.thrift
thriftrw --no-recurse --plugin="yarpc --sanitize-tchannel" --out=encoding/thrift/thriftrw-plugin-yarpc/internal/tests encoding/thrift/thriftrw-plugin-yarpc/internal/tests/weather.thrift

thrift-gen --generateThrift --outputDir internal/crossdock/thrift/gen-go --inputFile internal/crossdock/thrift/echo.thrift
thrift-gen --generateThrift --outputDir internal/crossdock/thrift/gen-go --inputFile internal/crossdock/thrift/gauntlet_tchannel.thrift | strip_thrift_warnings

thrift --gen go:thrift_import=github.com/apache/thrift/lib/go/thrift --out internal/crossdock/thrift/gen-go internal/crossdock/thrift/gauntlet_apache.thrift | strip_thrift_warnings

protoc_go yarpcproto/yarpc.proto
protoc_go_grpc internal/examples/protobuf/examplepb/example.proto
protoc_yarpc_go internal/examples/protobuf/examplepb/example.proto
protoc_go_grpc internal/crossdock/crossdockpb/crossdock.proto
protoc_yarpc_go internal/crossdock/crossdockpb/crossdock.proto
protoc_go_grpc encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing.proto
protoc_yarpc_go encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing.proto
protoc_go_grpc encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing_no_service.proto
protoc_yarpc_go encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing_no_service.proto
protoc_go_grpc internal/examples/streaming/stream.proto
protoc_yarpc_go internal/examples/streaming/stream.proto

ragel -Z -G2 -o internal/interpolate/parse.go internal/interpolate/parse.rl
gofmt -s -w internal/interpolate/parse.go
generated_by_ragel internal/interpolate/parse.go

touch internal/crossdock/thrift/gen-go/echo/.nocover
touch internal/crossdock/thrift/gen-go/gauntlet_apache/.nocover
touch internal/crossdock/thrift/gen-go/gauntlet_tchannel/.nocover
touch yarpcproto/.nocover

rm -rf internal/crossdock/thrift/gen-go/gauntlet_apache/second_service-remote # generated and not needed
rm -rf internal/crossdock/thrift/gen-go/gauntlet_apache/thrift_test-remote # generated and not needed

etc/bin/update-licenses.sh
etc/bin/generate-cover-ignore.sh

rm -f .dockerignore
cat .gitignore | sed 's/^\///' > .dockerignore
