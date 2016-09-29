package grpc

import (
	"strings"

	"github.com/yarpc/yarpc-go/transport"
	"google.golang.org/grpc/metadata"
)

// headerMapper converts gRCP Metadata to and from transport headers.
type headerMapper struct{ Prefix string }

var (
	applicationHeaders = headerMapper{ApplicationHeaderPrefix}
)

// toGRPCMetadata converts transport headers into gRPC Metadata.
//
// Headers are read from 'from' and written to 'to'. The final header collection
// is returned.
//
// If 'to' is nil, a new map will be assigned.
func (hm headerMapper) ToGRPCMetadata(from transport.Headers, to metadata.MD) metadata.MD {
	if to == nil {
		to = make(metadata.MD, from.Len())
	}
	for k, v := range from.Items() {
		Headers(to).Add(hm.Prefix+k, v)
	}
	return to
}

// fromGRPCMetadata converts GRPC Metadata to transport headers.
//
// Headers are read from 'from' and written to 'to'. The final header collection
// is returned.
//
// If 'to' is nil, a new map will be assigned.
func (hm headerMapper) FromGRPCMetadata(from metadata.MD, to transport.Headers) transport.Headers {
	prefixLen := len(hm.Prefix)
	for k := range from {
		if strings.HasPrefix(k, hm.Prefix) {
			key := k[prefixLen:]
			to = to.With(key, Headers(from).Get(k))
		}
		// Note: undefined behavior for multiple occurrences of the same header
	}
	return to
}

// Headers is a convenience type for converting Header information safely
type Headers map[string][]string

// Add adds the key, value pair to the header.
// It appends to any existing values associated with key.
func (h Headers) Add(key, value string) {
	h[key] = append(h[key], value)
}

// Set sets the header entries associated with key to
// the single element value. It replaces any existing
// values associated with key.
func (h Headers) Set(key, value string) {
	h[key] = []string{value}
}

// Get gets the first value associated with the given key.
// If there are no values associated with the key, Get returns "".
// Get is a convenience method. For more complex queries,
// access the map directly.
func (h Headers) Get(key string) string {
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
func (h Headers) Del(key string) {
	delete(h, key)
}
