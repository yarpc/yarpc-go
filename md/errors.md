
# YARPC Errors

### DecodeForInboundResponseError

Indicates that the body for an inbound response (response for an outbound
request) was either unrecognizeable or had an invalid structure for the
encoding or the interface definition for the procedure.

- type: `yarpc.inbound.decode-response`
- message: `YARPC decode for inbound response error`

### DecodeHeadersForInboundResponseError

Indicates that the headers for the inbound response were unrecognizeable or
invalid for the encoding.

- type: `yarpc.inbound.decode-response-headers`
- message: `YARPC decode for inbound response headers error`

### EncodeForOutboundRequestError

Indicates that the body for an outbound request was invalid and could not be
sent for the encoding or the interface definition of the procedure.

- type: `yarpc.outbound.encode-request`
- message: `YARPC encode for outbound request error`

### EncodeHeadersForOutboundRequestError

Indicates that the headers for an outbound request were invalid for the
encoding.

- type: `yarpc.outbound.encode-request-headers`
- message: `YARPC encode for outbound request headers error`

### InvalidTtlForOutboundRequestError

Indicates that the `Context-TTL-MS` header for an outbound request could not be
encoded as an integer or that it was out of the range of possible durations.

- type: `yarpc.outbound.invalid-ttl`
- message: `YARPC invalid TTL for outbound request error`

### NoCallerForOutboundRequestError

Indicates that an outbound request could not be sent because it lacked a caller
property.

- type: `yarpc.outbound.no-caller`
- message: `YARPC no caller for outbound request error`

### NoProcedureForOutboundRequestError

Indicates that an outbound request could not be sent because it lacked a
non-zero-length procedure property.

- type: `yarpc.outbound.no-procedure`
- message: `YARPC no procedure for outbound request error`

### NoServiceForOutboundRequestError

Indicates that an outbound request could not be sent because it lacked a
non-zero-length service property.

- type: `yarpc.outbound.no-service`
- message: `YARPC no service for outbound request error`


### NoTtlForOutboundRequestError

Indicates that an outbound request could not be sent because it lacked a
`ttl` request property.

- type: `yarpc.outbound.no-ttl`
- message: `YARPC no TTL for outbound request error`

### NoTransportForOutboundRequestError

Indicates that an outbound request could not be sent because the request lacked
a corresponding transport.

- type: `yarpc.outbound.no-transport`
- message: `YARPC no transport for outbound request error`

----

## Bad Request

### DecodeForInboundRequestError

Indicates that the body for an inbound request was either unrecognizeable or
had an invalid structure for the encoding or the interface definition for the
procedure.

- type: `yarpc.bad-request.inbound.decode-request`
- message: `YARPC decode for inbound request error`
- HTTP status code: 400
- TChannel error code: `BadRequest`

### DecodeHeadersForInboundRequestError

Indicates that the headers for the inbound request were unrecognizeable or
invalid for the encoding.

- type: `yarpc.bad-request.inbound.decode-request-headers`
- message: `YARPC decode for inbound request headers error`
- HTTP status code: 400
- TChannel error code: `BadRequest`

### InvalidTtlForInboundRequestError

Indicates that the `Context-TTL-MS` header for an inbound request could not be
parsed as an integer or that it was out of the range of possible durations.

- type: `yarpc.bad-request.inbound.invalid-ttl`
- message: `YARPC invalid TTL for inbound request error`
- HTTP status code: 400
- TChannel error code: `BadRequest`

### NoCallerForInboundRequestError

Indicates that an inbound request could not be handled because it lacked a
non-zero-length `Rpc-Caller` HTTP header or TChannel `cn` transport header.

- type: `yarpc.bad-request.inbound.no-caller`
- message: `YARPC no caller for inbound request error`
- HTTP status code: 400
- TChannel error code: `BadRequest`

### NoHandlerForInboundRequestError

Indicates that an inbound request could not be handled because the receiving
service had no registered handler for the requested service and procedure.

- type: `yarpc.bad-request.inbound.no-handler`
- message: `YARPC no handler for inbound request error`
- HTTP status code: 400
- TChannel error code: `BadRequest`

### NoProcedureForInboundRequestError

Indicates that an inbound request could not be handled because it lacked a
non-zero-length `Rpc-Procedure` HTTP header or TChannel `arg1`.

- type: `yarpc.bad-request.inbound.no-procedure`
- message: `YARPC no procedure for inbound request error`
- HTTP status code: 400
- TChannel error code: `BadRequest`

### NoTtlForInboundRequestError

Indicates that an inbound request could not be handled because it lacked
a `Context-TTL-MS` HTTP header (where `Context-` is the configured RPC context
propagation HTTP header prefix). This error cannot be effected over TChannel
since `ttl` is intrinsic to a call request frame.

- type: `yarpc.bad-request.inbound.no-ttl`
- message: `YARPC no TTL for inbound request error`
- HTTP status code: 400
- TChannel error code: `BadRequest`

### NoServiceForInboundRequestError

Indicates that an inbound request could not be handled because it lacked a
`Rpc-Service` HTTP header or TChannel `sn` transport message property.

- type: `yarpc.bad-request.inbound.no-service`
- message: `YARPC no service for inbound request error`
- HTTP status code: 400
- TChannel error code: `BadRequest`

## Bad Response

### EncodeForOutboundResponseError

Indicates that the body for an outbound response was invalid and could not be
sent for the encoding or the interface definition of the procedure.

- type: `yarpc.bad-response.outbound.encode-response`
- message: `YARPC encode for outbound response error`
- HTTP status code: 500
- TChannel error code: `UnexpectedRequest`

### EncodeHeadersForOutboundResponseError

Indicates that the headers for an outbound response were invalid for the
encoding.

- type: `yarpc.bad-response.outbound.encode-response-headers`
- message: `YARPC encode for outbound response headers error`
- HTTP status code: 500
- TChannel error code: `UnexpectedRequest`
