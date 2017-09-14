# YARPC Properties

This document describes the properties of YARPC contexts, requests, and
responses over both HTTP and TChannel and how they are captured in-process,
sufficient for YARPC peers to communicate.
This is a living document that will grow to include additional optional
request, response, and context properties.
This does not address how YARPC will reconcile communication with non-YARPC
HTTP or TChannel peers, though YARPC will expect the same information to be
conveyed in-process by a well-defined mapping for each individual case,
acknowledging that no two REST interfaces are alike.


## caller

The calling service name for routing. For TChannel, the calling service name is
expressed as the `cn` transport header. For HTTP, the caller name must be
expressed with the `Rpc-Caller` header.

YARPC services require this header for inbound requests.

- `request.caller` Python
- `request.caller` JavaScript
- `request.caller` Java
- `Request.Caller` Go

## service

The callee service name for routing. For TChannel, the service name is
expressed as the `sn` frame property. For HTTP, the service name may be
expressed with the `Rpc-Service` header.

YARPC services require the service name property on requests, to ensure that
intermediate proxies and routers can route the request to an instance of that
service.

Each service may implement multiple Thrift IDL services, so there is no direct
association.
Thrift "services" are a misnomer for "interfaces" and a single YARPC "service"
often will implement multiple such Thrift interfaces, and many YARPC services
will implement the same interface, e.g., the Meta interface.

- `request.service` Python
- `request.service` JavaScript
- `request.service` Java
- `Request.Service` Go

## procedure

The RPC method name for routing and also useful in a low-cardinality index for
tracing.
For TChannel, the procedure is expressed as the `arg1` string in UTF8.
For HTTP, the procedure must be expressed with the `Rpc-Procedure` header.

The procedure is expressed in Thrift IDL by the Thrift service name (distinct
from the service name for routing) and the method name, delimited by two colons
`::`. For the following example, the procedure is `Users::getUser`.

```thrift
struct User {
    1: required UUID uuid
}
service Users {
    User getUser(1: User user)
}
```

It is not sufficient to infer the procedure from an HTTP method and path
because the path may contain parameters.
Inferring a procedure and query parameters from the various styles of HTTP API
routers is out of scope for YARPC, which uses `POST` and request bodies
exclusively.

- `request.procedure` Python
- `request.procedure` JavaScript
- `request.procedure` Java
- `Request.Procedure` Go

## ttl / deadline

To avoid clock synchronization problems, on the wire, the ttl is expressed as a
relative duration like 100 for 100 milliseconds.

The ttl or "time to live" is a budget in milliseconds that remain before a
request times out.

In TChannel, the ttl is expressed with the `ttl` frame
property. Over HTTP, the ttl is expressed with the `Context-TTL-MS` header, with
the number of remaining milliseconds expressed as a string.

The default time to live is 30 seconds. Any expression of an alternate time to
live can only diminish this budget.

In Thrift IDL, the default time to live for a procedure can be expressed with
the `ttlms` annotation, measured in integer milliseconds.

```thrift
service Users {
    User getUser(1: User user) ( ttlms = '100' )
}
```

When a service receives an inbound request, it captures the current time and
then computes a deadline by adding the time to live.
When a service creates a dependent outbound request, it computes the outbound
request time to live by subtracting the current time from the deadline.
It may then also truncate the time to live to the time to live intrinsic to the
outbound request procedure, as expressed at the call site or in the Thrift IDL.

- `request.context.ttl` Python (as a `timedelta`)
- `request.context.ttl` JavaScript (measured in milliseconds)
- `request.context.ttl` Java (as a `org.joda.time.Duration)
- `context.Context` deadline in Go


## encoding

The request body can be encoded as a raw buffer (not encoded), JSON, or Thrift
(at time of writing).
The encoding is intrinsic to the procedure and must be expressed informatively
with the HTTP `Rpc-Encoding` header or the TChannel `as` request transport header.

Valid values for the encoding, and their corresponding HTTP `Content-Type`
header are:

| Rpc-Encoding   | Content-Type               |
|----------------|----------------------------|
| `raw`          | `application/octet-stream` |
| `json`         | `application/json`         |
| `thrift`       | `application/x-thrift`     |

The `Content-Type` header is informative. YARPC peers do not use this header.
The header is only useful to non-YARPC HTTP agents, provided that they have a
means to display data for the content type rather than download it to a file.


## response status

RPC errors are either transport or application errors.
Request handlers express application errors for messages they receive that may
have exceptional response cases.
The response body contains the exception, as expressed by the encoding.
The response status expresses whether the body contains an exceptional value or
a success value.

Over HTTP, the `Rpc-Status` header takes the value `success` or `error`.
If absent, `success` is the default status.

Over TChannel, the application error bit of an error frame expresses the
response status.


## response error

> The response error feature is not yet implemented in YARPC Go.

For transport errors, the `Rpc-Error` HTTP header expresses the name of the
error case, corresponding to the name of the corresponding TChannel error code.
The error message appears in the body of an HTTP error, and in the message
field of a TChannel error frame.

For application errors, the `Rpc-Error` header expresses the name of the
error case as a string intrinsic to the procedure.
An RPC caller must handle unknown error cases for forward compatibility.

For Thrift, the response error name is not present on the wire because it is
implied by the symbolication of the result field number to its name through the
IDL for that procedure, so the response name serves to provide a human-readable
error class for logging, stats, and legacy callers.

For JSON and raw, the `Rpc-Error` header is the only means to distinguish
error classes, and over HTTP and its absence is the only means to distinguish a
success response body.
For TChannel, application errors are also indicated by a non-zero code on a
call response frame.


## headers

YARPC request and response contexts may carry arbitrary, case-insenstive
headers.
The YARPC headers key-value store enforces case insensitivity and reserved
header names.

Every YARPC request, request context, response, and response context, carries
headers.
The serialization of headers depends on the transport and encoding as specified
in the [Headers][headers.md] document.

- `request.context.headers` Python (implementing the dict interface)
- `request.context.headers` JavaScript (implementing get and set)
- `request.context.headers` Java (implementing the Map interface)
- `Request.Ctx.Headers` Go (implementing Get and Set)

YARPC distinguishes context headers from request and response headers by virtue
of a prefix.
The default prefix is `Context-`.
The prefix exists only on the wire, not in the headers store.
YARPC propagates all request context headers from an inbound request to a
dependent outbound request implicitly.

YARPC provides a means to merge inbound response contexts with inbound request
contexts, particularly for ensuring causal ordering of dependent requests and
propagating through responses.

By default, when merging the response context of an incoming response with the
incoming request context, YARPC will implicitly merge any context headers from
the response context over the request context, taking the latter if there is a
collision.

YARPC reserves the ability to provide more semantically appropriate merge
semantics for other headers in future versions, provided that those semantics
gracefully degrade to the default behavior.


## shard key

A shard key is a string that hints at which instance of a cluster should handle
the request. The domain of the shard key is particular to the service.

Over TChannel, the `sk` transport header captures the shard key.

Over HTTP, the `Rpc-Shard-Key` header captures the shard key.


## routing key

The optional routing key overrides the service name to address the request to a
more specific group of tasks.
Routing keys are useful in combination with traffic policies that route
requests to different traffic groups based on availability and failover policies.

Over TChannel, the `rk` transport header captures the routing key.

Over HTTP, the `Rpc-Routing-Key` header captures the routing key.


## routing delegate

The optional routing delegate overrides the routing key and service name, to
address the request to a proxy service with application-specific routing
behavior.

The TChannel, the `rd` transport header captures the routing delegate.

Over HTTP, the `Rpc-Routing-Delegate` header captures the routing delegate.


## tracing / baggage

YARPC does not yet submit a convention for tracing nor baggage propagation.
Both of these concerns, YARPC leaves to an Open Tracing implementation.

<!--

not specified for YARPC Go v1:

## via / obo for tracing a request through routing delegates while preserving
   the original caller, to advise circut breakers and rate limiters
## depth / max hops
## auth token
## auth parameters
## retry
## speculate
## clocks
## call site marker
## request groups based on expected latency: high/low leader/follower

-->
