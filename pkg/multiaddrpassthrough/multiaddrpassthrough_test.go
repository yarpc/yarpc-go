package multiaddrpassthrough

import (
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

var _ resolver.ClientConn = (*testClientConn)(nil)

type testClientConn struct {
	target  string
	State   resolver.State
	mu      sync.Mutex
	addrs   []resolver.Address // protected by mu
	updates int                // protected by mu
	t       *testing.T
}

func (t *testClientConn) ParseServiceConfig(string) *serviceconfig.ParseResult {
	return nil
}

func (t *testClientConn) ReportError(error) {
}

func (t *testClientConn) UpdateState(state resolver.State) error {
	t.State = state
	return nil
}

func (t *testClientConn) NewAddress(addrs []resolver.Address) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.addrs = addrs
	t.updates++
}

func (t *testClientConn) getAddress() ([]resolver.Address, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.addrs, t.updates
}

// This shouldn't be called by our code since we don't support this.
func (t *testClientConn) NewServiceConfig(serviceConfig string) {
	assert.Fail(t.t, "unexpected call to NewServiceConfig")
	return
}

func TestParseTarget(t *testing.T) {

	tests := []struct {
		msg       string
		target    resolver.Target
		addrsWant []resolver.Address
		errWant   error
	}{
		{
			msg:       "Single IPv4",
			target:    resolver.Target{URL: url.URL{Host: "1.2.3.4:1234"}},
			addrsWant: []resolver.Address{{Addr: "1.2.3.4:1234"}},
		},
		{
			msg:       "Single IPv6",
			target:    resolver.Target{URL: url.URL{Host: "[2607:f8b0:400a:801::1001]:9000"}},
			addrsWant: []resolver.Address{{Addr: "[2607:f8b0:400a:801::1001]:9000"}},
		},
		{
			msg:    "Testing multiple IPv4s",
			target: resolver.Target{URL: url.URL{Host: "1.2.3.4:1234/5.6.7.8:1234"}},
			addrsWant: []resolver.Address{
				{Addr: "1.2.3.4:1234"},
				{Addr: "5.6.7.8:1234"},
			},
		},
		{
			msg:    "Mixed IPv6 and IPv4",
			target: resolver.Target{URL: url.URL{Host: "[2607:f8b0:400a:801::1001]:9000/[2607:f8b0:400a:801::1002]:2345/127.0.0.1:4567"}},
			addrsWant: []resolver.Address{
				{Addr: "[2607:f8b0:400a:801::1001]:9000"},
				{Addr: "[2607:f8b0:400a:801::1002]:2345"},
				{Addr: "127.0.0.1:4567"},
			},
		},
		{
			msg:     "Empty target",
			target:  resolver.Target{URL: url.URL{Host: ""}},
			errWant: errMissingAddr,
		},
		{
			msg:    "Localhost",
			target: resolver.Target{URL: url.URL{Host: "localhost:1000"}},
			addrsWant: []resolver.Address{
				{Addr: "localhost:1000"},
			},
		},
		{
			msg:    "IPv4 missing port",
			target: resolver.Target{URL: url.URL{Host: "999.1.1.1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			gotAddr, gotErr := parseTarget(tt.target)

			if tt.errWant != nil {
				assert.EqualError(t, gotErr, tt.errWant.Error())
			}
			assert.ElementsMatch(t, tt.addrsWant, gotAddr)
		})
	}
}

func TestBuild(t *testing.T) {
	tests := []struct {
		msg        string
		target     resolver.Target
		watAddress []resolver.Address
		wantErr    string
	}{
		{
			msg:        "IPv6",
			target:     resolver.Target{URL: url.URL{Host: "[2001:db8::1]:http"}},
			watAddress: []resolver.Address{{Addr: "[2001:db8::1]:http"}},
		},
		{
			msg:     "Empty address",
			target:  resolver.Target{URL: url.URL{Host: "127.0.0.1:12345/"}},
			wantErr: errMissingAddr.Error(),
		},
		{
			msg:     "Empty target",
			target:  resolver.Target{URL: url.URL{Host: ""}},
			wantErr: errMissingAddr.Error(),
		},
	}

	builder := &multiaddrPassthroughBuilder{}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {

			cc := &testClientConn{target: tt.target.URL.Host}
			gotResolver, gotError := builder.Build(tt.target, cc, resolver.BuildOptions{})
			if tt.wantErr != "" {
				assert.EqualError(t, gotError, tt.wantErr)
			} else {
				assert.ElementsMatch(t, cc.State.Addresses, tt.watAddress)
				gotResolver.Close()
			}
		})
	}
}
