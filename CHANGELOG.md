Releases
========

v0.4.0 (unreleased)
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
-   Add `OnewayHandler` and `HandlerSpec` to support oneway handlers.
    Transport inbounds can choose which RPC types to accept


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
