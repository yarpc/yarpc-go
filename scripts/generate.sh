#!/bin/bash

set -euo pipefail

# Regular expression for file paths that don't need licenses.
#
# We need to ignore internal/tests for licenses so that the golden test for
# thriftrw-plugin-yarpc can verify the contents of the generated code without
# running updateLicenses on it.
LICENSE_FILTER="/internal/tests/"

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

start_waitpids() {
  WAITPIDS=
}

do_waitpid() {
  $@ &
  WAITPIDS="${WAITPIDS} $!"
}

reset_waitpids() {
  for waitpid in ${WAITPIDS}; do
    wait "${waitpid}" || exit 1
  done
  WAITPIDS=
}

start_waitpids

do_waitpid mockgen -destination=api/peer/peertest/list.go -package=peertest go.uber.org/yarpc/api/peer Chooser,List
do_waitpid mockgen -destination=api/peer/peertest/peer.go -package=peertest go.uber.org/yarpc/api/peer Identifier,Peer
do_waitpid mockgen -destination=api/peer/peertest/transport.go -package=peertest go.uber.org/yarpc/api/peer Transport,Subscriber
do_waitpid mockgen -destination=api/transport/transporttest/clientconfig.go -package=transporttest go.uber.org/yarpc/api/transport ClientConfig,ClientConfigProvider
do_waitpid mockgen -destination=api/transport/transporttest/handler.go -package=transporttest go.uber.org/yarpc/api/transport UnaryHandler,OnewayHandler
do_waitpid mockgen -destination=api/transport/transporttest/inbound.go -package=transporttest go.uber.org/yarpc/api/transport Inbound
do_waitpid mockgen -destination=api/transport/transporttest/outbound.go -package=transporttest go.uber.org/yarpc/api/transport UnaryOutbound,OnewayOutbound
do_waitpid mockgen -destination=api/transport/transporttest/router.go -package=transporttest go.uber.org/yarpc/api/transport Router,RouteTable
do_waitpid mockgen -destination=encoding/thrift/mock_protocol_test.go -package=thrift go.uber.org/thriftrw/protocol Protocol
do_waitpid mockgen -destination=transport/x/redis/redistest/client.go -package=redistest go.uber.org/yarpc/transport/x/redis Client

do_waitpid generate_stringer ConnectionStatus ./api/peer
do_waitpid generate_stringer Type ./api/transport

do_waitpid thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/echo.thrift
do_waitpid thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/oneway.thrift
do_waitpid thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/gauntlet.thrift
do_waitpid thriftrw --plugin=yarpc --out=internal/examples/thrift-oneway internal/examples/thrift-oneway/sink.thrift
do_waitpid thriftrw --plugin=yarpc --out=internal/examples/thrift-hello/hello internal/examples/thrift-hello/hello/echo.thrift
do_waitpid thriftrw --plugin=yarpc --out=internal/examples/thrift-keyvalue/keyvalue internal/examples/thrift-keyvalue/keyvalue/kv.thrift
do_waitpid thriftrw --out=encoding/thrift encoding/thrift/internal.thrift
do_waitpid thriftrw --out=serialize serialize/internal.thrift

reset_waitpids

thriftrw --no-recurse --plugin=yarpc --out=encoding/thrift/thriftrw-plugin-yarpc/internal/tests encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common.thrift
thriftrw --no-recurse --plugin=yarpc --out=encoding/thrift/thriftrw-plugin-yarpc/internal/tests encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic.thrift

thrift-gen --generateThrift --outputDir internal/crossdock/thrift/gen-go --inputFile internal/crossdock/thrift/echo.thrift
thrift-gen --generateThrift --outputDir internal/crossdock/thrift/gen-go --inputFile internal/crossdock/thrift/gauntlet_tchannel.thrift

thrift --gen go:thrift_import=github.com/apache/thrift/lib/go/thrift --out internal/crossdock/thrift/gen-go internal/crossdock/thrift/gauntlet_apache.thrift

touch internal/crossdock/thrift/gen-go/echo/.nocover
touch internal/crossdock/thrift/gen-go/gauntlet_apache/.nocover
touch internal/crossdock/thrift/gen-go/gauntlet_tchannel/.nocover

# this must come at the end
python scripts/updateLicense.py $(go list -json $(glide nv) | \
	jq -r '.Dir + "/" + (.GoFiles | .[]) | select(test("'"$LICENSE_FILTER"'") | not)')
rm -rf internal/crossdock/thrift/gen-go/gauntlet_apache/second_service-remote # generated and not needed
rm -rf internal/crossdock/thrift/gen-go/gauntlet_apache/thrift_test-remote # generated and not needed
