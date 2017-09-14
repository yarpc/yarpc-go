# yarpc [![GoDoc][doc-img]][doc] [![GitHub release][release-img]][release] [![Mit License][mit-img]][mit] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov] [![Go Report Card][report-card-img]][report-card]

A message passing platform for Go that lets you:

* Write servers and clients with various encodings, including [JSON](http://www.json.org/), [Thrift](https://thrift.apache.org/), and [Protobuf](https://developers.google.com/protocol-buffers/).
* Expose servers over many transports simultaneously, including [HTTP/1.1](https://www.w3.org/Protocols/rfc2616/rfc2616.html), [gRPC](https://grpc.io/), and [TChannel](https://github.com/uber/tchannel).
* Migrate outbound calls between transports without any code changes using config.

## Installation

We recommend locking to [SemVer](http://semver.org/) range `^1` using [Glide](https://github.com/Masterminds/glide):

```
glide get 'go.uber.org/yarpc#^1'
```

## Usage

Explore working code in the [examples](internal/examples) package, or read the following guides:

| Material | Topics |
| :------- | :----- |
| [Introduction](.docs/01-introduction.md) | The problem area, key concepts, and important vocabulary. |
| [Getting Started](.docs/02-getting-started.md) | Writing basic services using the **HTTP** transport and **Raw** encoding. |
| [Adding Structure](.docs/03-adding-structure.md) | Loosely structured messages using the **JSON** encoding. |
| [Testing](.docs/04-testing.md) | Unit and integration tests, the recorder, and other test helpers |
| [Headers & Baggage](.docs/05-headers-and-baggage.md) | Propagating metadata between services. |
| [Middleware](.docs/06-middleware.md) | Shipping features across services, procedures, encodings, and transports. |
| [Errors](.docs/07-errors.md) | Understanding server and application errors. |
| [Binary Encodings](.docs/08-binary-encodings.md) | Strictly structured messages using the **Thrift** and **Protobuf** encodings. |
| [Configuring Transports](.docs/09-configuring-transports.md) | Supporting additional wire formats with the **gRPC** and **TChannel** transports. |
| [Custom Encodings](.docs/10-custom-encodings.md) | Exploring new serialization formats with a custom encoding. |
| [Custom Transports](.docs/11-custom-transports.md) | Exposing new wire formats without any code changes. |

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
