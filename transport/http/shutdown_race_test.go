// Copyright (c) 2026 Uber Technologies, Inc.
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

package http

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
)

// TestConcurrentReleasePeerAndStop releases isolated peers while the transport
// stops. A failed first dial leaves a started, empty pool, as happens when a
// destination is down. Registering a released pool's teardown must not race
// with Stop waiting for it.
func TestConcurrentReleasePeerAndStop(t *testing.T) {
	const peers = 16
	id := testIdentifier{"127.0.0.1:1"}
	sub := idSubscriber{1}
	for iteration := 0; iteration < 500; iteration++ {
		tr := NewTransport(DialContext(func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, errors.New("destination unavailable")
		}))
		if err := tr.Start(); err != nil {
			t.Fatal(err)
		}
		dialers := make([]*Dialer, peers)
		for i := range dialers {
			d := tr.NewDialer().WithConnectionIsolation()
			pi, err := d.RetainPeer(id, sub)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = pi.(*httpPeer).h2Sender()
			dialers[i] = d
		}
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(peers + 1)
		for _, d := range dialers {
			go func(d *Dialer) {
				defer wg.Done()
				<-start
				if err := d.ReleasePeer(id, sub); err != nil {
					t.Error(err)
				}
			}(d)
		}
		go func() {
			defer wg.Done()
			<-start
			if err := tr.Stop(); err != nil {
				t.Error(err)
			}
		}()
		close(start)
		wg.Wait()
	}
}
