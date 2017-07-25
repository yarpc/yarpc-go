# yarpc [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

A message passing platform for Go that:

* Supports multiple encodings like JSON, [Thrift](https://thrift.apache.org/), and [Protocol Buffers](https://developers.google.com/protocol-buffers/).
* Enables multiple transports like [HTTP/1.1](https://www.w3.org/Protocols/rfc2616/rfc2616.html), [gRPC](https://grpc.io/), and [TChannel](https://github.com/uber/tchannel).
* Allows messages to be sent directly and through queues like [Redis](https://redis.io/) and [Cherami](https://eng.uber.com/cherami/).

High-level features are implemented *above the wire*, enabling:

* Exposing a server over multiple transports simultaneously.
* Migrating outbound calls between transports with no code changes using config.
* Middleware that work even as the encoding and transport vary.

## Installation

We recommend locking to [SemVer](http://semver.org/) range `^1` using [Glide](https://github.com/Masterminds/glide):

```
glide get 'go.uber.org/yarpc#^1'
```

## Stability: Stable

This library follows [SemVer](http://semver.org/) strictly.

No breaking changes will be made to exported APIs before `v2.0.0` with the
exception of expiremental packages.

Experimental packages reside within packages named `x`, and are *not stable*. This means their
APIs can break at any time. The purpose of these packages is to incubate new features.
These packages are vetted with internal customers, and are moved out of
the `x` package when stable, at which point their public APIs will be locked.

[doc-img]: https://godoc.org/go.uber.org/yarpc?status.svg
[doc]: https://godoc.org/go.uber.org/yarpc
[ci-img]: https://travis-ci.org/yarpc/yarpc-go.svg?branch=dev
[cov-img]: https://codecov.io/gh/yarpc/yarpc-go/branch/dev/graph/badge.svg
[ci]: https://travis-ci.org/yarpc/yarpc-go
[cov]: https://codecov.io/gh/yarpc/yarpc-go/branch/dev
