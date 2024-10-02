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

var errMissingAddr = errors.New("missing address")

type multiaddrPassthroughBuilder struct{}

// Build creates and starts a multi address passthrough resolver.
// It expects the target to be a list of addresses on the format:
// multi-addr-passthrough:///192.168.0.1:2345/127.0.0.1:5678
func (*multiaddrPassthroughBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	addresses, err := parseTarget(target)
	if err != nil {
		return nil, err
	}

	r := &multiaddrPassthroughResolver{
		addresses: addresses,
		cc:        cc,
	}

	err = r.start()
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (*multiaddrPassthroughBuilder) Scheme() string {
	return _scheme
}

type multiaddrPassthroughResolver struct {
	addresses []resolver.Address
	cc        resolver.ClientConn
}

func (r *multiaddrPassthroughResolver) start() error {
	return r.cc.UpdateState(resolver.State{Addresses: r.addresses})
}

func (*multiaddrPassthroughResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (*multiaddrPassthroughResolver) Close() {}

func parseTarget(target resolver.Target) ([]resolver.Address, error) {
	addresses := []resolver.Address{}
	endpoints := strings.Split(target.URL.Host, "/")

	for _, endpoint := range endpoints {
		if endpoint == "" {
			return nil, errMissingAddr
		}

		host, port, err := net.SplitHostPort(endpoint)
		if err != nil {
			return nil, err
		}

		addresses = append(addresses, resolver.Address{Addr: net.JoinHostPort(host, port), Type: resolver.Backend})
	}

	return addresses, nil
}
