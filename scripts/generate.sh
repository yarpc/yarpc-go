#!/bin/bash

set -Ee

DIR="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

mockgen -destination=api/peer/peertest/list.go -package=peertest go.uber.org/yarpc/api/peer Chooser,List
mockgen -destination=api/peer/peertest/peer.go -package=peertest go.uber.org/yarpc/api/peer Identifier,Peer
mockgen -destination=api/peer/peertest/transport.go -package=peertest go.uber.org/yarpc/api/peer Transport,Subscriber
mockgen -destination=api/transport/transporttest/clientconfig.go -package=transporttest go.uber.org/yarpc/api/transport ClientConfig,ClientConfigProvider
mockgen -destination=api/transport/transporttest/handler.go -package=transporttest go.uber.org/yarpc/api/transport UnaryHandler,OnewayHandler
mockgen -destination=api/transport/transporttest/inbound.go -package=transporttest go.uber.org/yarpc/api/transport Inbound
mockgen -destination=api/transport/transporttest/outbound.go -package=transporttest go.uber.org/yarpc/api/transport UnaryOutbound,OnewayOutbound
mockgen -destination=api/transport/transporttest/router.go -package=transporttest go.uber.org/yarpc/api/transport Router,RouteTable
mockgen -destination=encoding/thrift/mock_protocol_test.go -package=thrift go.uber.org/thriftrw/protocol Protocol
mockgen -destination=encoding/thrift/mock_handler_test.go -package=thrift -source=encoding/thrift/register.go
mockgen -destination=transport/x/redis/redistest/client.go -package=redistest go.uber.org/yarpc/transport/x/redis Client
stringer -type=Type api/transport || true # this fails still
thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/echo.thrift
thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/oneway.thrift
thriftrw --plugin=yarpc --out=internal/crossdock/thrift internal/crossdock/thrift/gauntlet.thrift
thriftrw --plugin=yarpc --out=internal/examples/oneway internal/examples/oneway/sink.thrift
thriftrw --plugin=yarpc --out=internal/examples/oneway internal/examples/oneway/sink.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-hello/hello internal/examples/thrift-hello/hello/echo.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-hello/hello internal/examples/thrift-hello/hello/echo.thrift
thriftrw --plugin=yarpc --out=internal/examples/thrift-keyvalue/keyvalue internal/examples/thrift-keyvalue/keyvalue/kv.thrift
thriftrw --out=encoding/thrift encoding/thrift/internal.thrift
thriftrw --out=serialize serialize/internal.thrift
thrift-gen --generateThrift --outputDir internal/crossdock/thrift/gen-go --inputFile internal/crossdock/thrift/echo.thrift
thrift-gen --generateThrift --outputDir internal/crossdock/thrift/gen-go --inputFile internal/crossdock/thrift/gauntlet_tchannel.thrift
thrift --gen go:thrift_import=github.com/apache/thrift/lib/go/thrift --out internal/crossdock/thrift/gen-go internal/crossdock/thrift/gauntlet_apache.thrift
touch internal/crossdock/thrift/gen-go/echo/.nocover
touch internal/crossdock/thrift/gen-go/gauntlet_apache/.nocover
touch internal/crossdock/thrift/gen-go/gauntlet_tchannel/.nocover
python scripts/updateLicense.py $(go list -json $(glide nv) | jq -r '.Dir + "/" + (.GoFiles | .[])')
