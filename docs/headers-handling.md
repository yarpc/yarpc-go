# Headers handling

Yarpc has unified API for getting and setting headers. Although implementations may wary 
significantly from one transport to another.

This document describes details of headers handling in Yarpc.

# Existing behaviour

## HTTP

### Outbound - Request (writing via req.Headers.With)

All application headers are prepended with an 'Rpc-Header' prefix.

### Inbound - Request (Parsing)

Predefined list of headers is read and stripped from the inbound request.

Headers with prefix 'Rpc-Header-' will be forwarded to an application code (without prefix).

Only headers explicitly specified in the config will be passed to the application code as is. If header name doesn't have 'x-' prefix, 'header %s does not begin with 'x-'' message is returned.

### Inbound - Response (Writing)

All application headers are prepended with an 'Rpc-Header' prefix.

### Outbound - Response (Parsing)

Headers with prefix 'Rpc-Header-' will be forwarded to an application code (without prefix).

## TChannel

### Outbound - Request (writing via req.Headers.With)

Headers with any name may be added.

### Inbound - Request (Parsing)

Predefined list of headers (one header, actually) is read and stripped from the inbound request.

All other headers are forwarded as is to an application code.

### Inbound - Response (Writing)

Attempting to add a header with a name listed as reserved leads to an error "cannot use reserved header key".

### Outbound - Response (Parsing)

Headers with the names listed as reserved are deleted. All other headers are forwarded to an application code as is.

## GRPC

### Outbound - Request (writing via req.Headers.With)

Attempting to add headers with some of reserved names or with already set values lead to 'duplicate key' error.

Attempting to add headers with 'rpc-' prefix leads to 'cannot use reserved header in application headers' error.

### Inbound - Request (Parsing)

Predefined list of headers is read and stripped from the inbound request.

All other headers are forwarded as is to an application code.

### Inbound - Response (Writing)

Attempting to add headers with some of reserved names or already set values lead to 'duplicate key' error.

Attempting to add headers with 'rpc-' prefix leads to 'cannot use reserved header in application headers' error.

### Outbound - Response (Parsing)

Headers with 'rpc-' prefix will be omitted from forwarding to an application code.

# New behaviour

## HTTP, TChannel, GRPC

### Outbound - Request (writing via req.Headers.With) and Inbound - Response (Writing)

Attempting to add a header with a 'prc-' or '$rpc$-' prefixes leads to an error "cannot use reserved header key".

### Inbound - Request (Parsing) and Outbound - Response (Parsing)

Unparsed headers with 'rpc-' or '$rpc$-' prefixes ignored, i.e. not forwarded to an application code.
