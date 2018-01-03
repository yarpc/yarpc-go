// Copyright (c) 2018 Uber Technologies, Inc.
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

package types

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// NewPortProvider creates an object that can be used to synchronize ports in
// yarpctest infrastructure.  Ports can be acquired through the "Port" function
// which will create new ports for the test based on the id passed into the
// function.
func NewPortProvider(t testing.TB) *PortProvider {
	return &PortProvider{
		idToPort: make(map[string]*Port),
		t:        t,
	}
}

// PortProvider maintains a list of IDs to Ports.
type PortProvider struct {
	idToPort map[string]*Port
	t        testing.TB
}

// NamedPort will return a *Port object that exists for the passed in 'id', or
// it will create a *Port object if one does not already exist.
func (p *PortProvider) NamedPort(id string) *Port {
	port, ok := p.idToPort[id]
	if !ok {
		port = newPort(p.t)
		p.idToPort[id] = port
	}
	return port
}

func newPort(t testing.TB) *Port {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
	require.NoError(t, err)
	pieces := strings.Split(listener.Addr().String(), ":")
	port, err := strconv.ParseInt(pieces[len(pieces)-1], 10, 0)
	require.NoError(t, err)
	return &Port{
		Listener: listener,
		Port:     uint16(port),
	}
}

// Port is an option injectable primitive for synchronizing port numbers between
// requests and services.
type Port struct {
	api.NoopLifecycle
	Listener net.Listener
	Port     uint16
}

// Start is a Noop.
func (n *Port) Start(t testing.TB) error {
	return nil
}

// Stop closes the port's listener.
func (n *Port) Stop(t testing.TB) error {
	return n.Listener.Close()
}

// ApplyService implements api.ServiceOption.
func (n *Port) ApplyService(opts *api.ServiceOpts) {
	opts.Listener = n.Listener
	opts.Port = n.Port
}

// ApplyRequest implements api.RequestOption
func (n *Port) ApplyRequest(opts *api.RequestOpts) {
	opts.Port = n.Port
}

// ApplyClientStreamRequest implements ClientStreamRequestOption
func (n *Port) ApplyClientStreamRequest(opts *api.ClientStreamRequestOpts) {
	opts.Port = n.Port
}
