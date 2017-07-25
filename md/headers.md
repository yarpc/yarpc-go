
# YARPC Headers

YARPC supports headers in TChannel and HTTP.

*As a guiding principle, YARPC supports the intersection of expressible headers
across both HTTP and TChannel with the JSON, Thrift, and raw encodings.*

Headers are transported for both requests and responses.
Headers for both requests and responses are divided between the request and
response headers, and request and response _context_ headers.
YARPC does not support trailers.

- Headers are case-insensitive.
- Headers may have empty values.
- Headers must not have duplicate keys.
- Header keys and values are strings.

Regardless of the transport, request and context headers share a single key space.
Context headers are distinguished from request headers by virtue of a prefix.

The default context header prefix is `Context-`, and this prefix can be
overridden on the RPC constructor with the context prefix header option.
Separation of propagated context headers from non-propagated request and
response headers is the same for both TChannel and HTTP transports.

- Go: `ContextHeaderPrefix`
- Python: `context_header_prefix`
- Node.js: `contextHeaderPrefix`
- Java: `contextHeaderPrefix`

Since this prefix is configurable and applications must be agnostic to the
prefix, the prefix must exist only on the wire.  YARPC must remove the prefix
for the in-memory context headers store and reintroduce the prefix when writing
context headers to the wire.

TChannel with the JSON encoding transports all headers as a JSON object with
string values.
Consequently, for interoperability, there must not be duplicate headers.

HTTP transports all headers as HTTP headers.
Consequently, for interoperability, all headers must be case-insensitive.
All inbound headers must be normalized to lower-case.
All outbound headers must be sent with the given mixed-case on the wire.
All outbound headers may be normalized to lower-case to enforce the
no-duplicates rule.

An inbound request with duplicate, normalized keys is a bad request and effects
a [DecodeHeadersForInboundRequest](errors.md#DecodeHeadersForInboundRequest)
error.

An outbound request with duplicate, normalized keys cannot be sent and effects
an [EncodeHeadersForOutboundRequest](errors.md#EncodeHeadersForOutboundRequest)
error.
