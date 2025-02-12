// Copyright (c) 2025 Uber Technologies, Inc.
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

package noopresolver

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

func TestBuild(t *testing.T) {
	tests := []struct {
		msg        string
		target     resolver.Target
		watAddress []resolver.Address
		wantErr    string
	}{
		{
			msg:     "Non empty target",
			target:  resolver.Target{URL: url.URL{Path: "/[2001:db8::1]:http"}},
			wantErr: errInvalidTarget.Error(),
		},
		{
			msg:    "Empty target",
			target: resolver.Target{URL: url.URL{Path: ""}},
		},
	}

	builder := &noopBuilder{}
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
	wantAddr := []resolver.Address{}

	b := NewBuilder()

	cc := &testClientConn{}
	_, err := b.Build(resolver.Target{}, cc, resolver.BuildOptions{})
	assert.ElementsMatch(t, cc.State.Addresses, wantAddr, "Client connection received the wrong list of addresses")
	require.NoError(t, err, "unexpected error building the resolver")

	cc.failUpdate = true
	_, err = b.Build(resolver.Target{URL: url.URL{Path: dest}}, cc, resolver.BuildOptions{})
	require.Error(t, err)

}

func TestGRPCIntegration(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := grpc.NewServer()
	reflection.Register(s)
	t.Cleanup(s.GracefulStop)

	go func() {
		err := s.Serve(ln)
		require.NoError(t, err)
	}()

	b := NewBuilder()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = grpc.DialContext(ctx, b.Scheme()+":///"+ln.Addr().String(), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Error(t, err)

	_, err = grpc.DialContext(ctx, b.Scheme()+":///", grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Error(t, err, "expected to fail with deadline exceeded")
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
