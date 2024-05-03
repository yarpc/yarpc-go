# Headers handling

YARPC has a single unified API for getting and setting headers across three L7 protocols, although implementations may vary by a lot.

This document describes details of headers handling in Yarpc.

# Existing behaviour

Important to note that HTTP standard (for both plain HTTP and gRPC) supports multiple values for the same header.
Yarpc doesn't support this feature, uses only one value for each header. Sending two headers with the same name will result in undefined behaviour.

Another important note is, HTTP/gRPC header is case insensitive, but TChannel header is case sensitive. This is why we have an OriginalHeaders method on the call object (derived from `yarpc.CallFromContext(ctx)`).

## HTTP

### Outbound - Request (writing via req.Headers.With)

All application headers are prepended with an 'Rpc-Header' prefix.

### Inbound - Request (Parsing)

Predefined list of headers is read and stripped from the inbound request.

Headers with prefix 'Rpc-Header-' will be forwarded to the application handler (without prefix).

Only headers explicitly specified in the config will be passed to the application code as is. If header name doesn't have 'x-' prefix, 'header %s does not begin with 'x-'' message is returned.

### Inbound - Response (Writing)

All application headers are prepended with an 'Rpc-Header' prefix.

### Outbound - Response (Parsing)

Headers with prefix 'Rpc-Header-' will be forwarded to the application handler (without prefix).

## TChannel

### Outbound - Request (writing via req.Headers.With)

Headers with any name may be added.

### Inbound - Request (Parsing)

A single header (`rpc-caller-procedure`) is read and stripped from the inbound request.

All other headers are forwarded as is to the application handler.

### Inbound - Response (Writing)

Attempting to add a header with a name listed as [reserved by yarpc](../transport/tchannel/header.go#L60) leads to an error "cannot use reserved header key".

### Outbound - Response (Parsing)

Headers with the names listed as reserved are deleted. All other headers are forwarded to the application handler as is.

## GRPC

### Outbound - Request (writing via req.Headers.With)

Attempting to add headers with some of reserved names or with already set values lead to 'duplicate key' error.

Attempting to add headers with 'rpc-' prefix leads to 'cannot use reserved header in application headers' error.

### Inbound - Request (Parsing)

Predefined list of headers is read and stripped from the inbound request.

All other headers are forwarded as is to the application handler.

### Inbound - Response (Writing)

Attempting to add headers with some of reserved names or already set values lead to 'duplicate key' error.

Attempting to add headers with 'rpc-' prefix leads to 'cannot use reserved header in application headers' error.

### Outbound - Response (Parsing)

Headers with 'rpc-' prefix will be omitted from forwarding to the application handler.

# New behaviour

This behaviour implemented, but hidden behind the feature flag. Following metrics may be used to check affected edges:

Names: `grpc_reserved_headers_stripped`, `grpc_reserved_headers_error` with `"component": "yarpc-header-migration"` constant tag,
`source` and `dest` variable tags.

## HTTP, TChannel, GRPC

### Outbound - Request (writing via req.Headers.With) and Inbound - Response (Writing)

Attempting to add a header with a 'prc-' or '$rpc$-' prefixes leads to an error "cannot use reserved header key".

### Inbound - Request (Parsing) and Outbound - Response (Parsing)

Unparsed headers with 'rpc-' or '$rpc$-' prefixes ignored, i.e. not forwarded to the application handler.
