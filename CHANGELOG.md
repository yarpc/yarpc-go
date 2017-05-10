Releases
========

v1.9.0-dev (unreleased)
-------------------

-   http: Added support for configuring the HTTP transport using x/config.
-   tchannel: Added support for configuring the TChannel transport using
    x/config.


v1.8.0 (2017-05-01)
-------------------

-   Adds consistent structured logging and metrics to all RPCs. This feature
    may be enabled and configured through `yarpc.Config`.
-   Adds an `http.AddHeader` option to HTTP outbounds to send certain HTTP
    headers for all requests.
-   Options `thrift.Multiplexed` and `thrift.Enveloped` may now be provided for
    Thrift clients constructed by `yarpc.InjectClients` by adding a `thrift`
    tag to the corresponding struct field with the name of the option. See the
    Thrift package documentation for more details.
-   Adds support for matching and constructing `UnrecognizedProcedureError`s
    indicating that the router was unable to find a handler for the request.
-   Adds support for linking peer lists and peer updaters using the `peer.Bind`
    function.
-   Adds an accessor to Dispatcher which provides access to the inbound
    middleware used by that Dispatcher.
-   Fixes a bug where the TChannel inbounds would not write the response headers
    if the response body was empty.

Experimental:

-   x/config: The service name is no longer part of the configuration and must
    be passed as an argument to the `LoadConfig*` or `NewDispatcher*` methods.
-   x/config: Configuration structures may now annotate primitive fields with
    `config:",interpolate"` to support reading environment variables in them.
    See the `TransportSpec` documentation for more information.


v1.7.1 (2017-03-29)
-------------------

-   Thrift: Fixed a bug where deserialization of large lists would return
    corrupted data at high throughputs.


v1.7.0 (2017-03-20)
-----------------------

-   x/config adds support for a pluggable configuration system that allows
    building `yarpc.Config` and `yarpc.Dispatcher` objects from YAML and
    arbitrary `map[string]interface{}` objects. Check the package documentation
    for more information.
-	tchannel: mask existing procedures with provided procedures.
-   Adds a peer.Bind function that takes a peer.ChooserList and a binder
    (anything that binds a peer list to a peer provider and returns the
    Lifecycle of the binding), and returns a peer.Chooser that combines
    the lifecycle of the peer list and its bound peer provider.
    The peer chooser is suitable for passing to an outbound constructor,
    capturing the lifecycle of its dependencies.
-   Adds a peer.ChooserList interface to the API, for convenience when passing
    instances with both capabilities (suitable for outbounds, suitable for peer
    list updaters).


v1.6.0 (2017-03-08)
-----------------------

-   Remove buffer size limit from Thrift encoding/decoding buffer pool.
-   Increased efficiency of inbound/outbound requests by pooling buffers.
-   Added MaxIdleConnsPerHost option to HTTP transports.  This option will
    configure the number of idle (keep-alive) outbound connections the transport
    will maintain per host.
-   Fixed bug in Lifecycle Start/Stop where we would run the Stop functionality
    even if Start hadn't been called yet.
-   Updated RoundRobin and PeerHeap implementations to block until the list has
    started or a timeout had been exceeded.


v1.5.0 (2017-03-03)
-----------------------

-   Increased efficiency of Thrift encoding/decoding by pooling buffers.
-   x/yarpcmeta make it easy to expose the list of procedures and other
    introspection information of a dispatcher on itself.
-   Redis: `Client` now has an `IsRunning` function to match the `Lifecycle`
    interface.
-   TChannel: bug fix that allows a YARPC proxy to relay requests for any
    inbound service name. Requires upgrade of TChannel to version 1.4 or
    greater.


v1.4.0 (2017-02-14)
-----------------------

-   Relaxed version constraint for `jaeger-client-go` to `>= 1, < 3`.
-   TChannel transport now supports procedures with a different service name
    than the default taken from the dispatcher. This brings the TChannel
	transport up to par with HTTP.


v1.3.0 (2017-02-06)
-----------------------

-   Added a `tchannel.NewTransport`. The new transport, a replacement for the
    temporary `tchannel.NewChannelTransport`, supports YARPC peer choosers.

    ```go
    transport, err := tchannel.NewTransport(tchannel.ServiceName("keyvalue"))
    chooser := peerheap.New(transport)
    outbound := transport.NewOutbound(chooser)
    ```

    The new transport hides the implementation of TChannel entirely to give us
    flexibility going forward to relieve TChannel of all RPC-related
    responsibilities, leaving only the wire protocol at its core.
    As a consequence, you cannot thread an existing Channel through this
    transport.

-   All outbounds now support `Call` before `Start` and all peer choosers now
    support `Choose` before `Start`, within the context deadline.
    These would previously return an error indicating that the component was
    not yet started.  They now wait for the component to start, or for their
    deadline to expire.


v1.2.0 (2017-02-02)
-----------------------

-   Added heap based PeerList under `peer/x/peerheap`.
-   Added `RouterMiddleware` parameter to `yarpc.Config`, which, if provided,
    will allow customizing routing to handlers.
-   Added experimental `transports/x/cherami` for transporting RPCs through
    [Cherami](https://eng.uber.com/cherami/).
-   Added ability to specify a ServiceName for outbounds on the
    transport.Outbounds object.  This will allow defining outbounds with a
    `key` that is different from the service name they will use for requests.
    If no ServiceName is specified, the ServiceName will fallback to the
    config.Outbounds map `key`.

    Before:

    ```go
    config.Outbounds['service'] := transport.Outbounds{
        Unary: httpTransport.NewSingleOutbound(...)
    }
    ...
    cc := dispatcher.ClientConfig('service')
    cc.Service() // returns 'service'
    ```

    After (optional):

    ```go
    config.Outbounds['service-key'] := transport.Outbounds{
        ServiceName: 'service'
        Unary: httpTransport.NewSingleOutbound(...)
    }
    ...
    cc := dispatcher.ClientConfig('service-key')
    cc.Service() // returns 'service'
    ```


v1.1.0 (2017-01-24)
-----------------------

-   Thrift: Mock clients compatible with gomock are now generated for each
    service inside a test subpackage. Disable this by passing a `-no-gomock`
    flag to the plugin.


v1.0.1 (2017-01-11)
-------------------

-   Thrift: Fixed code generation for empty services.
-   Thrift: Fixed code generation for Thrift services that inherit other Thrift
    services.


v1.0.0 (2016-12-30)
-------------------

-   Stable release: No more breaking changes will be made in the 1.x release
    series.


v1.0.0-rc5 (2016-12-30)
-----------------------

-   **Breaking**: The ThriftRW plugin now generates code under the subpackages
    `${service}server` and `$[service}client` rather than
    `yarpc/${service}server` and `yarpc/${service}client`.

    Given a `kv.thrift` that defines a `KeyValue` service, previously the
    imports would be,

        import ".../kv/yarpc/keyvalueserver"
        import ".../kv/yarpc/keyvalueclient"

    The same packages will now be available at,

        import ".../kv/keyvalueserver"
        import ".../kv/keyvalueclient"

-   **Breaking**: `NewChannelTransport` can now return an error upon
    construction.
-   **Breaking**: `http.URLTemplate` has no effect on `http.NewSingleOutbound`.
-   `http.Transport.NewOutbound` now accepts `http.OutboundOption`s.


v1.0.0-rc4 (2016-12-28)
-----------------------

-   **Breaking**: Removed the `yarpc.ReqMeta` and `yarpc.ResMeta` types. To
    migrate your handlers, simply drop the argument and the return value from
    your handler definition.

    Before:

    ```go
    func (h *myHandler) Handle(ctx context.Context, reqMeta yarpc.ReqMeta, ...) (..., yarpc.ResMeta, error) {
        // ...
    }
    ```

    After:

    ```go
    func (h *myHandler) Handle(ctx context.Context, ...) (..., error) {
        // ...
    }
    ```

    To access information previously available in the `yarpc.ReqMeta` or to
    write response headers, use the `yarpc.CallFromContext` function.

-   **Breaking**: Removed the `yarpc.CallReqMeta` and `yarpc.CallResMeta`
    types. To migrate your call sites, drop the argument and remove the return
    value.

    Before:

    ```go
    res, resMeta, err := client.Call(ctx, reqMeta, ...)
    ```

    After:

    ```go
    res, err := client.Call(ctx, ...)
    ```

    Use `yarpc.CallOption`s to specify per-request options and
    `yarpc.ResponseHeaders` to receive response headers for the call.

-   **Breaking**: Removed `yarpc.Headers` in favor of `map[string]string`.
-   **Breaking**: `yarpc.Dispatcher` no longer implements the
    `transport.Router` interface.
-   **Breaking**: Start and Stop for Inbound and Outbound are now expected to
    be idempotent.
-   **Breaking**: Combine `ServiceProcedure` and `Registrant` into `Procedure`.
-   **Breaking**: Rename `Registrar` to `RouteTable`.
-   **Breaking**: Rename `Registry` to `Router`.
-   **Breaking**: Rename `middleware.{Oneway,Unary}{Inbound,Outbound}Middleware`
    to `middleware.{Oneway,Unary}{Inbound,Outbound}`
-   **Breaking**: Changed `peer.List.Update` to accept a `peer.ListUpdates`
    struct instead of a list of additions and removals
-   **Breaking**: yarpc.NewDispatcher now returns a pointer to a
    yarpc.Dispatcher. Previously, yarpc.Dispatcher was an interface, now a
    concrete struct.

    This change will allow us to extend the Dispatcher after the 1.0.0 release
    without breaking tests depending on the rigidity of the Dispatcher
    interface.
-   **Breaking**: `Peer.StartRequest` and `Peer.EndRequest` no longer accept a
    `dontNotify` argument.
-   Added `yarpc.IsBadRequestError`, `yarpc.IsUnexpectedError` and
    `yarpc.IsTimeoutError` functions.
-   Added a `transport.InboundBadRequestError` function to build errors which
    satisfy `transport.IsBadRequestError`.
-   Added a `transport.ValidateRequest` function to validate
    `transport.Request`s.


v1.0.0-rc3 (2016-12-09)
-----------------------

-   Moved the `yarpc/internal/crossdock/` and `yarpc/internal/examples`
    folders to `yarpc/crossdock/` and `yarpc/examples` respectively.

-   **Breaking**: Relocated the `go.uber.org/yarpc/transport` package to
    `go.uber.org/yarpc/api/transport`.  In the process the `middleware`
    logic from transport has been moved to `go.uber.org/yarpc/api/middleware`
    and the concrete implementation of the Registry has been moved from
    `transport.MapRegistry` to `yarpc.MapRegistry`.  This did **not** move the
    concrete implementations of http/tchannel from the `yarpc/transport/` directory.

-   **Breaking**: Relocated the `go.uber.org/yarpc/peer` package to
    `go.uber.org/yarpc/api/peer`. This does not include the concrete
    implementations still in the `/yarpc/peer/` directory.

-   **Breaking**: This version overhauls the code required for constructing
    inbounds and outbounds.

    Inbounds and Outbounds now share an underlying Transport, of which there
    should be one for each transport protocol, so one HTTP Transport for all
    HTTP inbounds and outbounds, and a TChannel transport for all TChannel
    inbounds and outbounds.

    Before:

    ```go
    ch, err := tchannelProper.NewChannel("example-service", nil)
    if err != nil {
        log.Fatalln(err)
    }
    yarpc.NewDispatcher(yarpc.Config{
        Name: "example-service",
        yarpc.Inbounds{
            http.NewInbound(":80"),
            tchannel.NewInbound(ch, tchannel.ListenAddr(":4040")),
        },
        yarpc.Outbounds{
            http.NewOutbound("http://example-service/rpc/v1"),
            tchannel.NewOutbound(ch, tchannel.HostPort("127.0.0.1:4040")),
        },
    })
    ```

    After:

    ```go
    httpTransport := http.NewTransport()
    tchannelTransport := tchannel.NewChannelTransport(
		tchannel.ServiceName("example-service"),
		tchannel.ListenAddr(":4040"),
    )
    yarpc.NewDispatcher(yarpc.Config{
        Name: "example-service",
        yarpc.Inbounds{
            httpTransport.NewInbound(":80"),
            tchannelTransport.NewInbound(),
        },
        yarpc.Outbounds{
            httpTransport.NewSingleOutbound("http://example-service/rpc/v1"),
            tchannelTransport.NewSingleOutbound("127.0.0.1:4040"),
        },
    })
    ```

    The dispatcher now collects all of the unique transport instances from
    inbounds and outbounds and manages their lifecycle independently.

    This version repurposed the name `NewOutbound` for outbounds with a peer
    chooser, whereas `NewSingleOutbound` is a convenience for creating an
    outbound addressing a specific single peer.
    You may need to rename existing usage. The compiler will complain that
    strings are not `peer.Chooser` instances.

    This version introduces support for peer choosers, peer lists, and peer
    list updaters for HTTP outbounds. This is made possible by the above
    change that introduces a concrete instance of a Transport for each
    protocol, which deduplicates peer instances across all inbounds and
    outbounds, making connection sharing and load balancing possible,
    eventually for all transport protocols.

    Note that we use `NewChannelTransport`, as opposed to `NewTransport`.
    We reserve this name for a future minor release that will provide
    parity with HTTP for outbounds with peer choosers.

    The new ChannelTransport constructor can still use a shared TChannel
    Channel instance, if that is required.

    ```go
    ch, err := tchannelProper.NewChannel("example-service", nil)
    if err != nil {
        log.Fatalln(err)
    }
    tchannelTransport := tchannel.NewChannelTransport(
        tchannel.WithChannel(ch),
		tchannel.ServiceName("example-service"),
		tchannel.ListenAddr(":4040"),
    )
    yarpc.NewDispatcher(yarpc.Config{
        Name: "example-service",
        yarpc.Inbounds{
            tchannelTransport.NewInbound(),
        },
    })
    ```

-   **Breaking**: the `transport.Inbound` and `transport.Outbound` interfaces
    now implement `Start()` without any arguments.

    The dispatcher no longer threads a dependencies object through the start
    method of every configured transport. The only existing dependency was an
    opentracing Tracer, which you can now thread through Transport constructor
    options instead.

    Before:

    ```go
    yarpc.NewDispatcher(yarpc.Config{
        yarpc.Inbounds{
            http.NewInbound(...),
        },
        yarpc.Outbounds{
            "callee": http.NewOutbound(...)
        },
        Tracer: opentracing.GlobalTracer(),
    })
    ```

    Now:

    ```go
    tracer := opentracing.GlobalTracer()
    httpTransport := http.NewTransport(
        http.Tracer(tracer),
    )
    tchannelTransport := tchannel.NewChannelTransport(
        tchannel.Tracer(tracer),
    )
    yarpc.NewDispatcher(yarpc.Config{
        Name: "example-service",
        yarpc.Inbounds{
            httpTransport.NewInbound(":80"),
            tchannelTransport.NewInbound(),
        },
        yarpc.Outbounds{
            httpTransport.NewSingleOutbound("http://example-service/rpc/v1"),
            tchannelTransport.NewSingleOutbound("127.0.0.1:4040"),
        },
    })
    ```

    The `yarpc.Config` `Tracer` property is still accepted, but unused and
    deprecated.

    The dispatcher no longer provides a `transport.ServiceDetail` as an
    argument to `Start` on inbound transports.  The `transport.ServiceDetail`
    no longer exists.  You no longer need to provide the service name to start
    an inbound, only a registry.  Instead of passing the service detail to start,
    the dispatcher now calls `inbound.SetRegistry(transport.Registry)` before
    calling `Start()`.

    Custom transport protocols must change their interface accordingly to
    satisfy the `transport.Inbound` interface.  Uses that construct inbounds
    manually must either call `SetRegistry` or use the `WithRegistry` chained
    configuration method before calling `Start` without a `ServiceDetail`.

    Before:

    ```go
    inbound := tchannel.NewInbound(...)
    err := inbound.Start(
        transport.ServiceDetail{
            Name: "service",
            Registry: registry,
        },
        transport.NoDeps,
    )
    ```

    Now:

    ```go
    transport := tchannel.NewTransport()
    inbound := transport.NewInbound()
    inbound.SetRegistry(registry)
    err := inbound.Start()
    ```

    The `transport.Deps` struct and `transport.NoDeps` instance no longer exist.

-   **Breaking**: TChannel inbound and outbound constructors now return
    pointers to Inbound and Outbound structs with private state satisfying the
    `transport.Inbound` and `transport.Outbound` interfaces.  These were
    previously transport specific Inbound and Outbound interfaces.
    This eliminates unnecessary polymorphism in some cases.

-   Introduced OpenTracing helpers for transport authors.
-   Created the `yarpc.Serialize` package for marshalling RPC messages at rest.
    Useful for transports that persist RPC messages.
-   Tranports have access to `DispatchOnewayHandler` and `DispatchUnaryHandler`.
    These should be called by all `transport.Inbounds` instead of directly
    calling handlers.

v1.0.0-rc2 (2016-12-02)
-----------------------

-   **Breaking** Renamed `Agent` to `Transport`.
-   **Breaking** Renamed `hostport.Peer`'s `AddSubscriber/RemoveSubscriber`
    to `Subscribe/Unsubscribe`.
-   **Breaking** Updated `Peer.StartRequest` to take a `dontNotify` `peer.Subscriber` to exempt
    from updates.  Also added `Peer.EndRequest` function to replace the `finish` callback
    from `Peer.StartRequest`.
-   **Breaking** Renamed `peer.List` to `peer.Chooser`, `peer.ChangeListener` to `peer.List`
    and `peer.Chooser.ChoosePeer` to `peer.Chooser.Choose`.
-   Reduced complexity of `single` `peer.Chooser` to retain the passed in peer immediately.
-   **Breaking** Moved `/peer/list/single.go` to `/peer/single/list.go`.
-   **Breaking** Moved `/peer/x/list/roundrobin.go` to `/peer/x/roundrobin/list.go`.
-   HTTP Oneway requests will now process http status codes and returns appropriate errors.
-   **Breaking** Update `roundrobin.New` function to stop accepting an initial peer list.
    Use `list.Update` to initialize the peers in the list instead.
-   **Breaking**: Rename `Channel` to `ClientConfig` for both the dispatcher
    method and the interface. `mydispatcher.Channel("myservice")` becomes
    `mydispatcher.ClientConfig("myservice")`. The `ClientConfig` object can
    then used to build a new Client as before:
    `NewMyThriftClient(mydispatcher.ClientConfig("myservice"))`.
-   A comment is added atop YAML files generated by the recorder to help
    understanding where they come from.

v1.0.0-rc1 (2016-11-23)
-----------------------

-   **Breaking**: Rename the `Interceptor` and `Filter` types to
    `UnaryInboundMiddleware` and `UnaryOutboundMiddleware` respectively.
-   **Breaking**: `yarpc.Config` now accepts middleware using the
    `InboundMiddleware` and `OutboundMiddleware` fields.

    Before:

        yarpc.Config{Interceptor: myInterceptor, Filter: myFilter}

    Now:

        yarpc.Config{
            InboundMiddleware: yarpc.InboundMiddleware{Unary: myInterceptor},
            OutboundMiddleware: yarpc.OutboundMiddleware{Unary: myFilter},
        }

-   Add support for Oneway middleware via the `OnewayInboundMiddleware` and
    `OnewayOutboundMiddleware` interfaces.


v0.5.0 (2016-11-21)
-------------------

-   **Breaking**: A detail of inbound transports has changed.
    Starting an inbound transport accepts a ServiceDetail, including
    the service name and a Registry. The Registry now must
    implement `Choose(context.Context, transport.Request) (HandlerSpec, error)`
    instead of `GetHandler(service, procedure string) (HandlerSpec, error)`.
    Note that in the prior release, `Handler` became `HandleSpec` to
    accommodate oneway handlers.
-   Upgrade to ThriftRW 1.0.
-   TChannel: `NewInbound` and `NewOutbound` now accept any object satisfying
    the `Channel` interface. This should work with existing `*tchannel.Channel`
    objects without any changes.
-   Introduced `yarpc.Inbounds` to be used instead of `[]transport.Inbound`
    when configuring a Dispatcher.
-   Add support for peer lists in HTTP outbounds.


v0.4.0 (2016-11-11)
-------------------

This release requires regeneration of ThriftRW code.

-   **Breaking**: Procedure registration must now always be done directly
    against the `Dispatcher`. Encoding-specific functions `json.Register`,
    `raw.Register`, and `thrift.Register` have been deprecated in favor of
    the `Dispatcher.Register` method. Existing code may be migrated by running
    the following commands on your go files.

    ```
    gofmt -w -r 'raw.Register(d, h) -> d.Register(h)' $file.go
    gofmt -w -r 'json.Register(d, h) -> d.Register(h)' $file.go
    gofmt -w -r 'thrift.Register(d, h) -> d.Register(h)' $file.go
    ```
-   Add `yarpc.InjectClients` to automatically instantiate and inject clients
    into structs that need them.
-   Thrift: Add a `Protocol` option to change the Thrift protocol used by
    clients and servers.
-   **Breaking**: Remove the ability to set Baggage Headers through yarpc, use
    opentracing baggage instead
-   **Breaking**: Transport options have been removed completely. Encoding
    values differently based on the transport is no longer supported.
-   **Breaking**: Thrift requests and responses are no longer enveloped by
    default. The `thrift.Enveloped` option may be used to turn enveloping on
    when instantiating Thrift clients or registering handlers.
-   **Breaking**: Use of `golang.org/x/net/context` has been dropped in favor
    of the standard library's `context` package.
-   Add support for providing peer lists to dynamically choose downstream
    peers in HTTP Outbounds
-   Rename `Handler` interface to `UnaryHandler` and separate `Outbound`
    interface into `Outbound` and `UnaryOutbound`.
-   Add `OnewayHandler` and `HandlerSpec` to support oneway handlers.
    Transport inbounds can choose which RPC types to accept
-   The package `yarpctest.recorder` can be used to record/replay requests
    during testing. A command line flag (`--recorder=replay|append|overwrite`)
    is used to control the mode during the execution of the test.


v0.3.1 (2016-09-31)
-------------------

-   Fix missing canonical import path to `go.uber.org/yarpc`.


v0.3.0 (2016-09-30)
-------------------

-   **Breaking**: Rename project to `go.uber.org/yarpc`.
-   **Breaking**: Switch to `go.uber.org/thriftrw ~0.3` from
    `github.com/thriftrw/thriftrw-go ~0.2`.
-   Update opentracing-go to `>= 1, < 2`.


v0.2.1 (2016-09-28)
-------------------

-   Loosen constraint on `opentracing-go` to `>= 0.9, < 2`.


v0.2.0 (2016-09-19)
-------------------

-   Update thriftrw-go to `>= 0.2, < 0.3`.
-   Implemented a ThriftRW plugin. This should now be used instead of the
    ThriftRW `--yarpc` flag. Check the documentation of the
    [thrift](https://godoc.org/github.com/yarpc/yarpc-go/encoding/thrift)
    package for instructions on how to use it.
-   Adds support for [opentracing][]. Pass an opentracing instance as a
    `Tracer` property of the YARPC config struct and both TChannel and HTTP
    transports will submit spans and propagate baggage.
-   This also modifies the public interface for transport inbounds and
    outbounds, which must now accept a transport.Deps struct. The deps struct
    carries the tracer and may eventually carry other dependencies.
-   Panics from user handlers are recovered. The panic is logged (stderr), and
    an unexpected error is returned to the client about it.
-   Thrift clients can now make requests to multiplexed Apache Thrift servers
    using the `thrift.Multiplexed` client option.

[opentracing]: http://opentracing.io/


v0.1.1 (2016-09-01)
-------------------

-   Use `github.com/yarpc/yarpc-go` as the import path; revert use of
    `go.uber.org/yarpc` vanity path. There is an issue in Glide `0.11` which
    causes installing these packages to fail, and thriftrw `~0.1`'s yarpc
    template is still using `github.com/yarpc/yarpc-go`.


v0.1.0 (2016-08-31)
-------------------

-   Initial release.
