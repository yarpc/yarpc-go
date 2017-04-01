# yarpc-go [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

With hundreds to thousands of services communicating with RPC, transport
protocols (like HTTP and [TChannel][]), encoding protocols (like JSON or
Thrift), and peer choosers are the concepts that vary year over year.
Separating these concerns allows services to change transports and wire
protocols without changing call sites or request handlers, build proxies and
wire protocol bridges, or experiment with load balancing strategies.
YARPC is a toolkit for services and proxies.

[TChannel]: https://github.com/uber/tchannel

YARPC breaks RPC into interchangeable encodings, transports, and peer
choosers.
YARPC for Go provides reference implementations for HTTP/1.1 and TChannel
transports, and also raw, JSON, and Thrift encodings.
YARPC for Go provides experimental implementations for a Redis transport, a
Protobuf encoding, and a round robin peer chooser.
YARPC for Go plans to provide a gRPC transport, and a load balancer that uses
a least-pending-requests strategy.
Peer choosers can implement any strategy, including load balancing or sharding,
in turn bound to any peer list updater, like an address file watcher.

Regardless of transport, every RPC has some common properties: caller name,
service name, procedure name, encoding name, deadline or TTL, headers, baggage
(multi-hop headers), and tracing.
Each RPC can also have an optional shard key, routing key, or routing delegate
for advanced routing.
YARPC transports use a shared API for capturing RPC metadata, so middleware can
apply to requests over any transport.

Each YARPC transport protocol can implement inbound handlers and outbound
callers. Each of these can support different RPC types, like unary (request and
response) or oneway (request and receipt) RPC. A future release of YARPC will
add support for other RPC types including variations on streaming and pubsub.


## Installation

```
go get -u go.uber.org/yarpc
```

If using [Glide](https://github.com/Masterminds/glide), *at least* `glide
version 0.12.3` is required to install:

```
$ glide --version
glide version 0.12.3

$ glide get 'go.uber.org/yarpc#^1'
```

To use Thrift code generation, you will need to install plugins.
These cannot be vendored since go depends on the binaries being available on
the path.

```
$ go get 'go.uber.org/thriftrw'
$ go get 'go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc'
```


## Examples

This [example][hello] illustrates a simple service that implements a handler
for a `Hello::echo` Thrift procedure.

```thrift
service Hello {
    EchoResponse echo(1:EchoRequest echo)
}

struct EchoRequest {
    1: required string message;
    2: required i16 count;
}

struct EchoResponse {
    1: required string message;
    2: required i16 count;
}
```

A go:generate directive informs `go generate` how to produce the Thrift models
and YARPC bindings for the echo service.

```go
//go:generate thriftrw --plugin=yarpc echo.thrift
```

```
$ go generate echo.thrift
```

Setting up a YARPC dispatcher configures inbounds and outbounds for supported
transport protocols and RPC types.
This sets a service up to receive HTTP requests on port 8080 and send requests
to itself.
YARPC funnels requests from all inbound transports into its routing table, and
organizes outbounds by name.

```go
httpTransport := http.NewTransport()
dispatcher := yarpc.NewDispatcher(yarpc.Config{
    Name: "hello",
    Inbounds: yarpc.Inbounds{
        httpTransport.NewInbound(":8080"),
    },
    Outbounds: yarpc.Outbounds{
        "hello": {
            Unary: httpTransport.NewSingleOutbound("http://127.0.0.1:8080"),
        },
    },
})
```

The dispatcher governs the lifecycle of every inbound, outbound, and the
singleton for each transport protocol.
The singleton can manage the lifecycle of shared peers and connections.

```go
if err := dispatcher.Start(); err != nil {
    log.Fatal(err)
}
defer dispatcher.Stop()
```

At the end of `main`, we block until we receive a signal to exit, then unravel
anything deferred like `dispatcher.Stop()`, shutting down gracefully.

```
signals := make(chan os.Signal, 1)
signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
<-signals
```

### Handle

To receive requests from any inbound, we register a handler object using the
Thrift generated server.

```go
dispatcher.Register(helloserver.New(&helloHandler{}))
```

The handler must implement the Hello service from the Thrift IDL.
The generated code requires a handler that implements `helloserver.Interface`.

```go
type helloHandler struct{}

func (h *helloHandler) Echo(ctx context.Context, e *echo.EchoRequest) (*echo.EchoResponse, error) {
	return &echo.EchoResponse{Message: e.Message, Count: e.Count + 1}, nil
}
```

### Call

To send a request on an outbound, we construct a client using the corresponding
named outbound from the dispatcher.
The client will use that name for the outbound request service name.

```go
client := helloclient.New(dispatcher.ClientConfig("hello"))
```

To call a remote procedure, the context *must* have a deadline.
We create a context with a one second deadline and call a method of the client.
The client will use the dispatcher name for the caller name, Thrift for the
encoding, and infer the procedure names from the `Echo` method (`Hello::echo`).

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

res, err := client.Echo(ctx, &echo.EchoRequest{Message: "Hello world", Count: 1})
if err != nil {
    log.Fatal(err)
}
fmt.Println(res)
```

### Other Examples

YARPC also provides examples for the [oneway][] RPC type and a key value
service using both the [Thrift][thrift-keyvalue] and [JSON][json-keyvalue]
encodings.

[hello]: https://github.com/yarpc/yarpc-go/tree/master/internal/examples/thrift-hello
[oneway]: https://github.com/yarpc/yarpc-go/tree/master/internal/examples/thrift-oneway
[thrift-keyvalue]: https://github.com/yarpc/yarpc-go/tree/master/internal/examples/thrift-keyvalue
[json-keyvalue]: https://github.com/yarpc/yarpc-go/tree/master/internal/examples/json-keyvalue

<!--
TODO
- headers
- tracing
- baggage
- middleware
- oneway
- custom encoding
- custom transport
- routing key
- routing delegate (route by tenancy baggage)
- handle-or-forward (route by shard key)
- transport bridge (http to tchannel)
- custom peer chooser-list (sharding)
- custom peer chooser-list (round robin for example)
- custom peer list updater (dns srv records)
-->

## Development Status: Stable

Ready for most users. No breaking changes to stable APIs will be made before
2.0.

Stable:
- handler and call sites for unary and oneway
- dispatcher constructor and config type
- transport constructors (including the `tchannel.NewTransportChannel(...)`
  although YARPC will eventually also have `tchannel.NewTransport(chooser,
  ...)`)
- interfaces for "go.uber.org/yarpc/api/transport" Transport (for lifecycle
  management), Inbound, Outbound, Request, Response, ResponseWriter, Router,
  RouteTable, Procedure, and Lifecycle
- interfaces for "go.uber.org/yarpc/api/peer" Transport (for peer management),
  Chooser, List
- the middleware API
- wire representation of RPC for HTTP and TChannel, including all required
  headers: Rpc-Caller, Rpc-Service, Rpc-Procedure, and Context-TTL-MS.

Unstable:
- Any package in an `x` directory, including the experimental Redis transport,
  the Protobuf encoding, and the round-robin peer chooser.
- debug and introspection APIs (these are internal to prevent external
  implementations of transports, inbounds, and outbounds from making use of
  them, but we further do not guarantee the content of debug pages)

Upcoming:
- peer choosers for TChannel
- handle-or-forward request handlers, possibly using per-procedure middleware
- streaming RPC type for some transports (gRPC, WebSocket)
- pubsub RPC type for some transports (Redis)

[doc-img]: https://godoc.org/go.uber.org/yarpc?status.svg
[doc]: https://godoc.org/go.uber.org/yarpc
[ci-img]: https://travis-ci.org/yarpc/yarpc-go.svg?branch=master
[cov-img]: https://coveralls.io/repos/github/yarpc/yarpc-go/badge.svg?branch=master
[ci]: https://travis-ci.org/yarpc/yarpc-go
[cov]: https://coveralls.io/github/yarpc/yarpc-go?branch=master
