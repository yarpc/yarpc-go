// Copyright (c) 2020 Uber Technologies, Inc.
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

package inboundbuffermiddleware

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap/zaptest"
)

func TestStartStop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	// Exercise the Time option just for the coverage.
	buffer := New(Time(time.Now))
	require.NoError(t, buffer.Start(ctx))
	for i := 0; i < 8; i++ {
		require.NoError(t, buffer.Handle(ctx, &transport.Request{Body: &bytes.Buffer{}}, &transporttest.FakeResponseWriter{}, transporttest.EchoHandler{}))
	}
	require.NoError(t, buffer.Stop(ctx))
}

type slowHandler struct{}

func (slowHandler) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	_, err := io.Copy(resw, req.Body)
	time.Sleep(time.Millisecond)
	return err
}

func TestStress(t *testing.T) {
	body := "Hello, World!\n\n"
	var handler slowHandler

	ctx, cancel := context.WithTimeout(context.Background(), 5*testtime.Second)
	defer cancel()

	buffer := New(Capacity(8), Concurrency(8), Logger(zaptest.NewLogger(t)))
	require.NoError(t, buffer.Start(ctx))

	concurrency := 64

	type result struct {
		errors       int
		drops        int
		corrupt      int
		missingDelay int
	}

	resCh := make(chan result)
	for client := 0; client < concurrency; client++ {
		go func(client int) {
			var res result

			for i := 0; i < 64; i++ {
				resw := &transporttest.FakeResponseWriter{}
				err := buffer.Handle(ctx, &transport.Request{
					Body: bytes.NewBufferString(body),
				}, resw, handler)

				if yarpcerrors.IsResourceExhausted(err) {
					res.drops++
				} else if err != nil {
					res.errors++
				} else if _, ok := resw.Headers.Get("buffer-delay-ns"); !ok {
					res.missingDelay++
				} else if resw.Body.String() != body {
					res.corrupt++
				}
			}

			resCh <- res
		}(client)
	}

	var res result
	for i := 0; i < concurrency; i++ {
		r := <-resCh
		res.errors += r.errors
		res.drops += r.drops
		res.corrupt += r.corrupt
	}

	require.NoError(t, buffer.Stop(ctx))
	assert.Zero(t, res.corrupt)
	assert.Zero(t, res.missingDelay)

	// Report
	t.Logf("%#v\n", res)
}
