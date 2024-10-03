package multiaddrpassthrough

import (
	"errors"
	"net"
	"strings"

	"google.golang.org/grpc/resolver"
)

func init() {
	resolver.Register(&multiaddrPassthroughBuilder{})
}

const _scheme = "multi-addr-passthrough"

var (
	errMissingAddr     = errors.New("missing address")
	errInvaildEndpoint = errors.New("specified endpoint is invalid")
)

type multiaddrPassthroughBuilder struct{}
type multiaddrPassthroughResolver struct{}

// NewBuilder creates a new multi address passthrough resolver builder.
func NewBuilder() resolver.Builder {
	return &multiaddrPassthroughBuilder{}
}

// Build creates and starts a multi address passthrough resolver.
// It expects the target to be a list of addresses on the format:
// multi-addr-passthrough:///192.168.0.1:2345/127.0.0.1:5678
func (*multiaddrPassthroughBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	addresses, err := parseTarget(target)
	if err != nil {
		return nil, err
	}

	err = cc.UpdateState(resolver.State{Addresses: addresses})
	if err != nil {
		return nil, err
	}

	return &multiaddrPassthroughResolver{}, nil
}

func (*multiaddrPassthroughBuilder) Scheme() string {
	return _scheme
}

// ResolveNow is a noop for the multi address passthrough resolver.
func (*multiaddrPassthroughResolver) ResolveNow(resolver.ResolveNowOptions) {}

// Close is a noop for the multi address passthrough resolver.
func (*multiaddrPassthroughResolver) Close() {}

func parseTarget(target resolver.Target) ([]resolver.Address, error) {
	endpoints := strings.Split(target.URL.Path, "/")
	addresses := make([]resolver.Address, 0, len(endpoints))

	for _, endpoint := range endpoints {
		if len(endpoint) > 0 {
			_, _, err := net.SplitHostPort(endpoint)
			if err != nil {
				return nil, errInvaildEndpoint
			}

			addresses = append(addresses, resolver.Address{Addr: endpoint})
		}
	}

	if len(addresses) == 0 {
		return nil, errMissingAddr
	}
	return addresses, nil
}
