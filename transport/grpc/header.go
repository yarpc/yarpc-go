package grpc

import (
	"strings"

	"go.uber.org/yarpc/transport"

	"google.golang.org/grpc/metadata"
)

// headerMapper converts gRPC Metadata to and from transport headers.
type headerMapper struct{ Prefix string }

var (
	applicationHeaders = headerMapper{ApplicationHeaderPrefix}
)

// toMetadata converts transport headers into gRPC Metadata.
//
// Headers are read from 'from' and written to 'to'. The final header collection
// is returned.
//
// If 'to' is nil, a new map will be assigned.
func (hm headerMapper) toMetadata(from transport.Headers, to metadata.MD) metadata.MD {
	if to == nil {
		to = make(metadata.MD, from.Len())
	}
	for k, v := range from.Items() {
		headers(to).add(hm.Prefix+k, v)
	}
	return to
}

// fromMetadata converts GRPC Metadata to transport headers.
//
// Headers are read from 'from' and written to 'to'. The final header collection
// is returned.
//
// If 'to' is nil, a new map will be assigned.
func (hm headerMapper) fromMetadata(from metadata.MD, to transport.Headers) transport.Headers {
	prefixLen := len(hm.Prefix)
	for k := range from {
		if strings.HasPrefix(k, hm.Prefix) {
			key := k[prefixLen:]
			to = to.With(key, headers(from).get(k))
		}
		// Note: undefined behavior for multiple occurrences of the same header
	}
	return to
}

// headers is a convenience type for converting Header information safely
type headers map[string][]string

// Add adds the key, value pair to the header.
// It appends to any existing values associated with key.
func (h headers) add(key, value string) {
	h[key] = append(h[key], value)
}

// Set sets the header entries associated with key to
// the single element value. It replaces any existing
// values associated with key.
func (h headers) set(key, value string) {
	h[key] = []string{value}
}

// Get gets the first value associated with the given key.
// If there are no values associated with the key, Get returns "".
// Get is a convenience method. For more complex queries,
// access the map directly.
func (h headers) get(key string) string {
	if h == nil {
		return ""
	}
	v := h[key]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// Del deletes the values associated with key.
func (h headers) del(key string) {
	delete(h, key)
}
