
# YARPC

YARPC is a coherent set of transport and encoding agnostic RPC libraries.
YARPC libraries in [Python][], [Node.js][], [Go][], and [Java][] ship with HTTP
and [TChannel][] transports and JSON and [Thrift][] encodings, and provides a
common core that can be used for these or alternate encodings and transports.
Each implementation provides a test harness and test subject as a Docker image
and uses [Crossdock][] to validate correctness and consistency.

[Python]: https://github.com/yarpc/yarpc-python
[Node.js]: https://github.com/yarpc/yarpc-node
[Go]: https://github.com/yarpc/yarpc-go
[Java]: https://github.com/yarpc/yarpc-java
[TChannel]: https://github.com/uber/tchannel
[Thrift]: https://github.com/thriftrw/
[Crossdock]: https://github.com/yarpc/crossdock
