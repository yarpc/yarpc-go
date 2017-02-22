#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

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

mockgen -destination=api/middleware/middlewaretest/router.go -package=middlewaretest go.uber.org/yarpc/api/middleware Router
mockgen -destination=api/peer/peertest/list.go -package=peertest go.uber.org/yarpc/api/peer Chooser,List
mockgen -destination=api/peer/peertest/peer.go -package=peertest go.uber.org/yarpc/api/peer Identifier,Peer
mockgen -destination=api/peer/peertest/transport.go -package=peertest go.uber.org/yarpc/api/peer Transport,Subscriber
mockgen -destination=api/transport/transporttest/clientconfig.go -package=transporttest go.uber.org/yarpc/api/transport ClientConfig,ClientConfigProvider
mockgen -destination=api/transport/transporttest/handler.go -package=transporttest go.uber.org/yarpc/api/transport UnaryHandler,OnewayHandler
mockgen -destination=api/transport/transporttest/inbound.go -package=transporttest go.uber.org/yarpc/api/transport Inbound
mockgen -destination=api/transport/transporttest/outbound.go -package=transporttest go.uber.org/yarpc/api/transport UnaryOutbound,OnewayOutbound
mockgen -destination=api/transport/transporttest/router.go -package=transporttest go.uber.org/yarpc/api/transport Router,RouteTable
mockgen -source=vendor/go.uber.org/thriftrw/protocol/protocol.go -destination=encoding/thrift/mock_protocol_test.go -package=thrift go.uber.org/thriftrw/protocol Protocol
mockgen -destination=transport/x/redis/redistest/client.go -package=redistest go.uber.org/yarpc/transport/x/redis Client

generate_stringer ConnectionStatus ./api/peer
generate_stringer LifecycleState ./internal/sync
generate_stringer Type ./api/transport

thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/echo.thrift
thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/oneway.thrift
thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/gauntlet.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-oneway internal/examples/thrift-oneway/sink.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-hello/hello internal/examples/thrift-hello/hello/echo.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-keyvalue/keyvalue internal/examples/thrift-keyvalue/keyvalue/kv.thrift
thriftrw --plugin=yarpc --out=transport/x/cherami/example/thrift transport/x/cherami/example/thrift/example.thrift
thriftrw --out=encoding/thrift encoding/thrift/internal.thrift
thriftrw --out=serialize serialize/internal.thrift

thriftrw --no-recurse --plugin=yarpc --out=encoding/thrift/thriftrw-plugin-yarpc/internal/tests encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common.thrift
thriftrw --no-recurse --plugin=yarpc --out=encoding/thrift/thriftrw-plugin-yarpc/internal/tests encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic.thrift

thrift-gen --generateThrift --outputDir internal/crossdock/thrift/gen-go --inputFile internal/crossdock/thrift/echo.thrift
thrift-gen --generateThrift --outputDir internal/crossdock/thrift/gen-go --inputFile internal/crossdock/thrift/gauntlet_tchannel.thrift

thrift --gen go:thrift_import=github.com/apache/thrift/lib/go/thrift --out internal/crossdock/thrift/gen-go internal/crossdock/thrift/gauntlet_apache.thrift

touch internal/crossdock/thrift/gen-go/echo/.nocover
touch internal/crossdock/thrift/gen-go/gauntlet_apache/.nocover
touch internal/crossdock/thrift/gen-go/gauntlet_tchannel/.nocover

scripts/updateLicenses.sh
