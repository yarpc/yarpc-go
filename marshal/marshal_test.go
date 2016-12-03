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

package marshal

import (
	"bytes"
	"testing"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestMarshaling(t *testing.T) {
	tracer := opentracing.NoopTracer{}
	spanContext := tracer.StartSpan("test-span").Context()

	headers := map[string]string{
		"hello": "world",
		"foo":   "bar",
	}
	body := []byte(
		"My mind is tellin' me no... but my body... my body's tellin' me yes",
	)

	haveReq := &transport.Request{
		Caller:          "Caller",
		Service:         "ServiceName",
		Encoding:        "Encoding",
		Procedure:       "Procedure",
		Headers:         transport.HeadersFromMap(headers),
		ShardKey:        "ShardKey",
		RoutingKey:      "RoutingKey",
		RoutingDelegate: "RoutingDelegate",
		Body:            bytes.NewReader(body),
	}

	matcher := transporttest.NewRequestMatcher(t, haveReq)

	marshalledReq, err := MarshalRPC(tracer, spanContext, haveReq)
	require.NoError(t, err, "could not marshal RPC to bytes")
	assert.NotEmpty(t, marshalledReq)

	_, gotReq, err := UnmarshalRPC(tracer, marshalledReq)
	require.NoError(t, err, "could not unmarshal RPC from bytes")

	assert.True(t, matcher.Matches(gotReq))
}
