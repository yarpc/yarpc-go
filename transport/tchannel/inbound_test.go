// Copyright (c) 2016 Uber Technologies, Inc.
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

package tchannel

import (
	"testing"

	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundStartNew(t *testing.T) {
	x, err := NewTransport(ServiceName("foo"))
	require.NoError(t, err)

	i := x.NewInbound()
	i.SetRouter(new(transporttest.MockRouter))
	require.NoError(t, i.Start())
	require.NoError(t, x.Start())
	require.NoError(t, i.Stop())
	require.NoError(t, x.Stop())
}

func TestInboundStopWithoutStarting(t *testing.T) {
	x, err := NewTransport(ServiceName("foo"))
	require.NoError(t, err)
	i := x.NewInbound()
	assert.NoError(t, i.Stop())
}

func TestInboundInvalidAddress(t *testing.T) {
	x, err := NewTransport(ServiceName("foo"), ListenAddr("not valid"))
	require.NoError(t, err)

	i := x.NewInbound()
	i.SetRouter(new(transporttest.MockRouter))
	assert.Nil(t, i.Start())
	defer i.Stop()
	assert.Error(t, x.Start())
	defer x.Stop()
}
