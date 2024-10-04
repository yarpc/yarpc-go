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
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
			target:    resolver.Target{URL: url.URL{Path: "1.2.3.4:1234"}},
			addrsWant: []resolver.Address{{Addr: "1.2.3.4:1234"}},
		}, {
			msg:       "Single IPv4, leading slash",
			target:    resolver.Target{URL: url.URL{Path: "/1.2.3.4:1234"}},
			addrsWant: []resolver.Address{{Addr: "1.2.3.4:1234"}},
		},
		{
			msg:       "Single IPv6",
			target:    resolver.Target{URL: url.URL{Path: "[2607:f8b0:400a:801::1001]:9000"}},
			addrsWant: []resolver.Address{{Addr: "[2607:f8b0:400a:801::1001]:9000"}},
		},
		{
			msg:    "Testing multiple IPv4s",
			target: resolver.Target{URL: url.URL{Path: "1.2.3.4:1234/5.6.7.8:1234"}},
			addrsWant: []resolver.Address{
				{Addr: "1.2.3.4:1234"},
				{Addr: "5.6.7.8:1234"},
			},
		},
		{
			msg:    "Mixed IPv6 and IPv4",
			target: resolver.Target{URL: url.URL{Path: "[2607:f8b0:400a:801::1001]:9000/[2607:f8b0:400a:801::1002]:2345/127.0.0.1:4567"}},
			addrsWant: []resolver.Address{
				{Addr: "[2607:f8b0:400a:801::1001]:9000"},
				{Addr: "[2607:f8b0:400a:801::1002]:2345"},
				{Addr: "127.0.0.1:4567"},
			},
		},
		{
			msg:     "Empty target",
			target:  resolver.Target{URL: url.URL{Path: ""}},
			errWant: errMissingAddr.Error(),
		},
		{
			msg:    "Localhost",
			target: resolver.Target{URL: url.URL{Path: "localhost:1000"}},
			addrsWant: []resolver.Address{
				{Addr: "localhost:1000"},
			},
		},
		{
			msg:     "Invalid IPv4",
			target:  resolver.Target{URL: url.URL{Path: "999.1.1.1"}},
			errWant: errInvaildEndpoint.Error(),
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
			target:     resolver.Target{URL: url.URL{Path: "[2001:db8::1]:http"}},
			watAddress: []resolver.Address{{Addr: "[2001:db8::1]:http"}},
		},
		{
			msg:     "Invalid target",
			target:  resolver.Target{URL: url.URL{Path: "127.0.0.1"}},
			wantErr: errInvaildEndpoint.Error(),
		},
		{
			msg:     "Empty target",
			target:  resolver.Target{URL: url.URL{Path: ""}},
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
	_, err := b.Build(resolver.Target{URL: url.URL{Path: dest}}, cc, resolver.BuildOptions{})
	assert.ElementsMatch(t, cc.State.Addresses, wantAddr, "Client connection received the wrong list of addresses")
	require.NoError(t, err, "unexpected error building the resolver")
}

func TestGRPCIntegration(t *testing.T) {
	dest := "127.0.0.1:3456/192.168.1.1:6789"

	b := NewBuilder()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := grpc.DialContext(ctx, b.Scheme()+":///"+dest, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
}

type testClientConn struct {
	target string
	State  resolver.State
	mu     sync.Mutex
	addrs  []resolver.Address // protected by mu
	t      *testing.T
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
}

// This shouldn't be called by our code since we don't support this.
func (t *testClientConn) NewServiceConfig(serviceConfig string) {
	assert.Fail(t.t, "unexpected call to NewServiceConfig")
}
