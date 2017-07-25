# HTTP Semantics

YARPC HTTP outbound requests must:

- Must use the `POST` method.

- Have a path `/yarpc/v1`.

- Must have a request content type of `application/json` for JSON and
  `application/octet-stream` for other encodings.

- Ok responses must have a content type of `application/json` for JSON and
  `application/octet-stream` for other encodings.

- Not ok responses must have a `text/plain` content type, and a response body
  containing the error message.

- The HTTP `Host` header as defined by RFC 2616 is required.

- All request headers must have the `Rpc-Header-` prefix.

- All context headers must have the `Context-` prefix.
  All context headers must be forwarded from request to response and from
  response back to request.
  The default merge semantics is to take the last header of all headers with
  equivalent names.
  Context headers may have special merge semantics.
  The default merge semantics must be an acceptable fallback for all special
  context headers, e.g., `Context-TTL-MS`.

- All other request properties must have the `Rpc-` prefix.

YARPC HTTP inbound request handlers may ignore the HTTP request method, path,
and content-type.


# Success and Error Semantics

## Success Semantics

All successes are alike, each kind of error expressed in its own way.

- HTTP: A 200 response status code.

- HTTP: An "OK" response status message.

- TChannel: "code" call response frame byte set to 0.

- The HTTP response body and TChannel response arg3 are the same.
  For Thrift, this is a result struct with a "success" field. For the raw
  encoding, this is a raw response body. For the JSON encoding, the body is a
  successful result, and unlike Thrift, there is no envelope indicating that it
  is successful.


## Application Errors

In other cases, the message is received by the application and the application
responds with an error case.

- HTTP: A 200 response status code.

- HTTP: An "OK" response status message.

- TChannel: "code" call response frame byte set to 1.

  Non-zero code does not imply anything about whether this request should be
  retried.

  Implementations should implement unix-style zero / non-zero logic to be
  future safe to other "not ok" codes.

  For Thrift, the error cases are expressed as part of the response body.  The
  Thrift result struct represents a union of field 0, implicitly named
  "success", with any number of other fields for each application error case.

- A `Rpc-Error` header with the name of the exception case.

  For Thrift, the name corresponds to the name of the exception field of the
  result struct, as expressed in the Thrift IDL and implied on the wire.
  However, the error name is still useful for logging and stats for RPC layers
  that are not privy to the IDL, and useful to clients with an IDL on hand that
  does not recognize new exceptional cases.

  For raw and JSON, the error name is not expressed in the response body, so
  this header is the sole means for a caller to switch on the exception case.

- HTTP response body and TChannel arg3:

  For Thrift, the body contains a result struct with a non-zero, non-success
  field describing the exception case.

  For JSON, the body is a JSON description of the exception, meaningful to the
  application.

  For the raw mode, the body is a free form buffer, meaningful to the
  application.


## Transport Errors

A transport error is any error that cannot be expressed in the response body.

As with [TChannel][TChannel Errors], YARPC will classify transport errors into
one of the following categories, expressed as the name of the TChannel error in
the `Rpc-Error` headers.
For example,

```
Rpc-Error: BadRequest
```

The `Rpc-Error` header is normative for determining retry, flow control,
congestion avoidance, and peer selection.
The HTTP status code is informative.
For a transport error, the response body is the informative plain-text error
message with a newline.
Clients must not depend on the text of an error programmatically.
Error text may change in minor releases.
Error responses must have the `Content-Type` header of `text/plain; charset=utf8`.

[TChannel Errors]: https://github.com/uber/tchannel/blob/master/docs/protocol.md#code1-1

| TC   | TChannel        | HTTP |
|------|-----------------|------|
| 0x01 | Timeout         | 500  |
| 0x02 | Cancelled       | 400  |
| 0x03 | Busy            | 400  |
| 0x04 | Declined        | 500  |
| 0x05 | UnexpectedError | 500  |
| 0x06 | BadRequest      | 400  |
| 0x07 | NetworkError    | 500  |
| 0x08 | ProtocolError   | 500  |
| 0x09 | Unhealthy       | 500  |

- `BadRequest`: The request body could not be decoded. This does not capture
  requests that did not pass domain validation—such exceptions must be
  expressed as encoded response classes.

  HTTP Status Code: 400 Bad Request.

- `Cancelled`: Caller aborted the request.

  HTTP Status Code: 400

- `Unhealthy`: A circuit broke, anywhere between the caller and the service. The
  request should not be retried.

  HTTP Status Code: 500

- `Timeout`: The request time to live elapsed.

  HTTP Status Code: 500

- `Busy`: Encountered a rate limit or work shedding, anywhere between the caller
  and the service.

  HTTP Status Code: 400

- `Declined`: Any entity between the caller and the service declined for reasons
  other than load and may be retried elsewhere without an adjustment to routing
  preferences.

  HTTP Status Code: 500

- `UnexpectedError`: The request may have been initiated, retry only if the
  procedure is idempotent or duplicate application can be handled.

  HTTP Status Code: 500

- `NetworkError`: The network failed to deliver the request. The request was
  never seen by the service and may be safely retried. A request that fails due
  to a broken connection qualifies as a network error.

  HTTP Status Code: 500

- `ProtocolError`: The request framing became malformed anywhere between the
  caller and the service, indicating corruption in transit or a flaw in
  marshalling. This error class captures cases like invalid HTTP header names
  and bad checksums.

  HTTP Status Code: 500
