
# YARPC Test Subject

This document describes the procedures that a YARPC test subject service must
serve.

The test subject must listen for inbound HTTP connections on 8081 and for
inbound TChannel connections on 8082.
The test subject may be run within a [test harness][Behaviors] that listens on
port 8080 for [Crossdock][] behavior requests.

[Crossdock]: https://github.com/yarpc/crossdock
[Behaviors]: crossdock.md

The test subject must use enveloped Thrift messages over HTTP and must use
un-enveloped Thrift messages over TChannel.

| Port | Transport | Role               |
|------|-----------|--------------------|
| 8081 | HTTP      | YARPC Test Subject |
| 8082 | TChannel  | YARPC Test Subject |
| 8083 | HTTP      | Forced Error Cases |
| 8084 | TChannel  | Forced Error Cases |
| 8085 | HTTP      | YARPC Third Party  |
| 8086 | TChannel  | YARPC Third Party  |
| 8087 | TChannel  | YARPC Context Hop  |
| 8088 | HTTP      | Apache Thrift      |

### echo/raw

Copies the request body to the response body.
Also copies the request headers to the response headers.

- procedure: `echo/raw`
- encoding: `raw`
- HTTP status code: 200


### echo

Copies the request body to the response body.
Also copies the request headers to the response headers.

- procedure: `echo`
- encoding: `json`
- HTTP status code: 200


### Echo::echo

Takes a ping request and returns a pong success struct, copying the beep from
the ping to the boop of the pong.
Also copies the request headers to the response headers.

- procedure: `Echo::echo`
- encoding: `thrift`
- HTTP status code: 200

YARPC must produce a [DecodeForInboundRequestError][] if the inbound request
body is not a valid Echo::echo arguments struct.

[DecodeForInboundRequestError]: errors.md#DecodeForInboundRequestError

```thrift
service Echo {
    Pong echo(1: Ping ping) (
        ttlms = '100'
    )
}

struct Ping {
    1: required string beep
}

struct Pong {
    1: required string boop
}
```


### hangup

Produces a transport error.

- procedure: `hangup`
- encoding: `json`
- HTTP status code: 500
- HTTP `Rpc-Error` header: `UnexpectedError`
- TChannel error name: `UnexpectedError`
- response body: `expected error`

Transport errors comprise all exceptional cases that cannot be expressed as a
valid application response body for the procedure.
So, supposing that a procedure produces an exception that cannot be expressed
with a response body that is valid for the procedure's response JSON schema,
YARPC must provide an unexpected error.
This endpoint produces such an error with the "expected error" message.


### Test::hangup

Produces a transport error with a Thrift application exception in its response
body envelope.

- procedure: `Test::hangup`
- encoding: `thrift`
- HTTP status code: 500
- HTTP `Rpc-Error` header: `UnexpectedError`
- TChannel error name: `UnexpectedError`
- response body: Thrift application exception envelope bearing an `UNKNOWN`
  error type and `expected error` message.

This procedure's behavior is only defined for inbounds that are configured to
support Thrift message envelopes.

*Legacy TChannel services do not use Thrift message envelopes since all of the
same information can and is expressed with TChannel transport headers.
The same applies for YARPC (all of the same information is expressed with
`Rpc-` prefixed response headers).
Thrift message envelopes exist solely for compatibility with legacy Apache
Thrift HTTP services.*


### error

Produces an application exception, defined in the domain of or schema for the
response body.

- procedure: `error`
- encoding: `json`
- HTTP status code: 500
- HTTP `Rpc-Error` header: `error`
- response body: `{"error": "yuno"}`

For Thrift, the `Rpc-Error` header captures the name of the exception case
(the name of the property on the result struct that corresponds to the `throws`
clause for that exception class).
For JSON, the exception name is not necessarily expressed in the response body,
so the HTTP header is necessary to distinguish exception classes.


### bad-response

Responds with an `EncodeForOutboundResponseError`.

- procedure: `bad-response`
- encoding: `json`
- HTTP status code: 500
- HTTP `Rpc-Error` header: `UnexpectedError`
- TChannel error name: `UnexpectedError`
- response body: `YARPC encode for outbound response error`

*non-normative: this behavior can be either induced through a low-level
interface, but preferably through the normal channel, for example, by
attempting to respond with a cyclic object graph.*.

```js
var cycle = {};
cycle.cycle = cycle;
callback(null, {body: cycle});
```


### never

Never responds.

- procedure: `never`
- encoding: `json`
