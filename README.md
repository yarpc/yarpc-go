# YARPC [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

A message passing platform for Go that:

* Supports multiple binary encodings like Thrift and Protocol Buffers.
* Enables multi-transport services with support for HTTP/1.1, gRPC, and TChannel.
* Allows messages to be sent directly and through queues like Redis and Cherami.

High-level features are implemented *above the wire*, enabling:

* Exposing a server over multiple transports simultaneously
* Migrating outbound calls between transports with no code changes using config
* Shipping a set of middleware that work regardless of encoding or transport

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
