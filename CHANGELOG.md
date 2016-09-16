Releases
========

v0.2.0 (unreleased)
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
