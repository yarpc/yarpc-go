# yarpc [![GoDoc][doc-img]][doc] [![GitHub release][release-img]][release] [![Mit License][mit-img]][mit] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov] [![Go Report Card][report-card-img]][report-card]

A message passing platform for Go that lets you:

* Write servers and clients with various encodings, including [JSON](http://www.json.org/), [Thrift](https://thrift.apache.org/), and [Protobuf](https://developers.google.com/protocol-buffers/).
* Expose servers over many transports simultaneously, including [HTTP/1.1](https://www.w3.org/Protocols/rfc2616/rfc2616.html), [gRPC](https://grpc.io/), and [TChannel](https://github.com/uber/tchannel).
* Migrate outbound calls between transports without any code changes using config.

## Getting Started

1. [Introduction](.docs/intro.md)
    * Installation
    * Understanding YARPC's architecture
    * RPC Types: Unary, Oneway, and Streaming
    * Choosing Transports
    * Glossary and Terms
2. [Basic Usage](.docs/basic-usage.md)
    * Implementing and testing handlers
    * Configuring inbound transports
    * Using Yab to call handlers
    * Writing and testing outbound calls
    * Configuring outbound transports
    * Available Transports
3. [Production Essentials](.docs/prod-essentials.md)
    * Understanding Errors
    * Graceful Shutdown
    * Introspection and Debug Pages
    * Tracing and Metrics
    * Retries and Circuit Breaking
    * Rate Limiting and SLAs
4. [Advanced Usage](.docs/advanced-usage.md)
    * Middleware
    * Proxies
6. [Extending YARPC](.docs/extending-yarpc.md)
    * Composing Outbounds
    * Authoring Custom Transports
    * Creating a Custom Encoding

## Stability

This library is `v1` and follows [SemVer](http://semver.org/) strictly.

No breaking changes will be made to exported APIs before `v2.0.0` with the
**exception of experimental packages**.

Experimental packages reside within packages named `x`, and are *not stable*. This means their
APIs can break at any time. The intention here is to validate these APIs and iterate on them
by working closely with internal customers. Once stable, their contents will be moved out of
the containing `x` package and their APIs will be locked.

[doc-img]: http://img.shields.io/badge/GoDoc-Reference-blue.svg
[doc]: https://godoc.org/go.uber.org/yarpc

[release-img]: https://img.shields.io/github/release/yarpc/yarpc-go.svg
[release]: https://github.com/yarpc/yarpc-go/releases

[mit-img]: http://img.shields.io/badge/License-MIT-blue.svg
[mit]: https://github.com/yarpc/yarpc-go/blob/master/LICENSE

[ci-img]: https://img.shields.io/travis/yarpc/yarpc-go/master.svg
[ci]: https://travis-ci.org/yarpc/yarpc-go/branches

[cov-img]: https://codecov.io/gh/yarpc/yarpc-go/branch/master/graph/badge.svg
[cov]: https://codecov.io/gh/yarpc/yarpc-go/branch/master

[report-card-img]: https://goreportcard.com/badge/go.uber.org/yarpc
[report-card]: https://goreportcard.com/report/go.uber.org/yarpc
