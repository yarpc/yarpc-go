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

package chooserbenchmark

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/pendingheap"
	"go.uber.org/yarpc/peer/roundrobin"
)

func TestNewContext(t *testing.T) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	defer func() {
		err = f.Close()
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	config := &Config{
		ClientGroups: []ClientGroup{
			{
				Name:  "roundrobin",
				Count: 5,
				RPS:   2,
				Constructor: func(t peer.Transport) peer.ChooserList {
					return roundrobin.New(t)
				},
			},
			{
				Name:  "pendingheap",
				Count: 5,
				RPS:   2,
				Constructor: func(t peer.Transport) peer.ChooserList {
					return pendingheap.New(t)
				},
			},
		},
		ServerGroups: []ServerGroup{
			{
				Name:    "normal",
				Count:   5,
				Latency: time.Millisecond * 1,
			},
		},
		Duration: 10 * time.Millisecond,
		Output:   f,
	}
	ctx, err := NewContext(config)
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Millisecond, ctx.Duration)
	assert.Equal(t, 10, len(ctx.Clients))
	assert.Equal(t, 5, len(ctx.Servers))
	assert.Equal(t, 5, len(ctx.Listeners))
}
