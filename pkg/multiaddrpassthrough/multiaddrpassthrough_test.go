// Copyright (c) 2024 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package multiaddrpassthrough

import (
	"context"
	"errors"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

var _ resolver.ClientConn = (*testClientConn)(nil)

func TestParseTarget(t *testing.T) {

	tests := []struct {
		msg       string
		target    resolver.Target
		addrsWant []resolver.Address
		errWant   string
	}{
		{
			msg:       "Single IPv4",
			target:    resolver.Target{Endpoint: "1.2.3.4:1234", URL: url.URL{Path: "/1.2.3.4:1234"}},
			addrsWant: []resolver.Address{{Addr: "1.2.3.4:1234"}},
		},
		{
			msg:       "Single IPv4, leading slash",
			target:    resolver.Target{Endpoint: "1.2.3.4:1234", URL: url.URL{Path: "/1.2.3.4:1234"}},
			addrsWant: []resolver.Address{{Addr: "1.2.3.4:1234"}},
		},
		{
			msg:       "Single IPv6",
			target:    resolver.Target{Endpoint: "[2607:f8b0:400a:801::1001]:9000", URL: url.URL{Path: "/[2607:f8b0:400a:801::1001]:9000"}},
			addrsWant: []resolver.Address{{Addr: "[2607:f8b0:400a:801::1001]:9000"}},
		},
		{
			msg:    "Multiple IPv4s",
			target: resolver.Target{Endpoint: "1.2.3.4:1234/5.6.7.8:1234", URL: url.URL{Path: "/1.2.3.4:1234/5.6.7.8:1234"}},
			addrsWant: []resolver.Address{
				{Addr: "1.2.3.4:1234"},
				{Addr: "5.6.7.8:1234"},
			},
		},
		{
			msg:    "Multiple IPv4s, double slash",
			target: resolver.Target{Endpoint: "1.2.3.4:1234//5.6.7.8:1234", URL: url.URL{Path: "/1.2.3.4:1234/5.6.7.8:1234"}},
			addrsWant: []resolver.Address{
				{Addr: "1.2.3.4:1234"},
				{Addr: "5.6.7.8:1234"},
			},
		},
		{
			msg:    "Mixed IPv6 and IPv4",
			target: resolver.Target{Endpoint: "[2607:f8b0:400a:801::1001]:9000/[2607:f8b0:400a:801::1002]:2345/127.0.0.1:4567", URL: url.URL{Path: "/[2607:f8b0:400a:801::1001]:9000/[2607:f8b0:400a:801::1002]:2345/127.0.0.1:4567"}},
			addrsWant: []resolver.Address{
				{Addr: "[2607:f8b0:400a:801::1001]:9000"},
				{Addr: "[2607:f8b0:400a:801::1002]:2345"},
				{Addr: "127.0.0.1:4567"},
			},
		},
		{
			msg:     "Empty target",
			target:  resolver.Target{Endpoint: "", URL: url.URL{Path: "/"}},
			errWant: errMissingAddr.Error(),
		},
		{
			msg:    "Localhost",
			target: resolver.Target{Endpoint: "localhost:1000", URL: url.URL{Path: "/localhost:1000"}},
			addrsWant: []resolver.Address{
				{Addr: "localhost:1000"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			gotAddr, gotErr := parseTarget(tt.target)

			if gotErr != nil {
				assert.EqualError(t, gotErr, tt.errWant)
			}
			assert.ElementsMatch(t, gotAddr, tt.addrsWant)
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
			target:     resolver.Target{Endpoint: "[2001:db8::1]:http", URL: url.URL{Path: "/[2001:db8::1]:http"}},
			watAddress: []resolver.Address{{Addr: "[2001:db8::1]:http"}},
		},
		{
			msg:     "Empty target",
			target:  resolver.Target{Endpoint: "", URL: url.URL{Path: "/"}},
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

func TestClientConnectionIntegration(t *testing.T) {
	dest := "127.0.0.1:3456"
	wantAddr := []resolver.Address{{Addr: dest}}

	b := NewBuilder()

	cc := &testClientConn{}
	_, err := b.Build(resolver.Target{Endpoint: dest, URL: url.URL{Path: dest}}, cc, resolver.BuildOptions{})
	assert.ElementsMatch(t, cc.State.Addresses, wantAddr, "Client connection received the wrong list of addresses")
	require.NoError(t, err, "unexpected error building the resolver")

	cc.failUpdate = true
	_, err = b.Build(resolver.Target{Endpoint: dest, URL: url.URL{Path: dest}}, cc, resolver.BuildOptions{})
	require.Error(t, err)

}

func TestGRPCIntegration(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := grpc.NewServer()
	reflection.Register(s)
	defer s.GracefulStop()

	go func() {
		err := s.Serve(ln)
		require.NoError(t, err)
	}()

	b := NewBuilder()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	conn, err := grpc.DialContext(ctx, b.Scheme()+":///"+ln.Addr().String(), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)

	defer func() {
		err := conn.Close()
		require.NoError(t, err)
	}()
}

type testClientConn struct {
	target     string
	failUpdate bool
	State      resolver.State
	mu         sync.Mutex
	addrs      []resolver.Address // protected by mu
	t          *testing.T
}

func (t *testClientConn) ParseServiceConfig(string) *serviceconfig.ParseResult {
	return nil
}

func (t *testClientConn) ReportError(error) {
}

func (t *testClientConn) UpdateState(state resolver.State) error {
	t.State = state
	if t.failUpdate {
		return errors.New("failed to update state")
	}
	return nil
}

func (t *testClientConn) NewAddress(addrs []resolver.Address) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.addrs = addrs
}

// This shouldn't be called by our code since we don't support this.
func (t *testClientConn) NewServiceConfig(serviceConfig string) {
	assert.Fail(t.t, "unexpected call to NewServiceConfig")
}
