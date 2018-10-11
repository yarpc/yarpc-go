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

package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var _ grpc.ServerStream = (*fakeServerStream)(nil)

type fakeServerStream struct {
	context context.Context
}

func (f *fakeServerStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeServerStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeServerStream) SetTrailer(metadata.MD)       {}
func (f *fakeServerStream) Context() context.Context     { return f.context }
func (f *fakeServerStream) SendMsg(m interface{}) error  { return nil }
func (f *fakeServerStream) RecvMsg(m interface{}) error  { return nil }

func TestInvalidStreamContext(t *testing.T) {
	ss := &fakeServerStream{context: context.Background()}

	_, err := requestFromServerStream(ss, "serv/proc")
	require.Contains(t, err.Error(), "cannot get metadata from ctx:")
	require.Contains(t, err.Error(), "code:internal")
}

func TestInvalidStreamMethod(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	ss := &fakeServerStream{context: ctx}

	_, err := requestFromServerStream(ss, "invalidMethod!")
	require.Contains(t, err.Error(), errInvalidGRPCMethod.Error())
}

func TestInvalidStreamRequest(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	ss := &fakeServerStream{context: ctx}

	_, err := requestFromServerStream(ss, "service/proc")
	require.Contains(t, err.Error(), "code:invalid-argument")
	require.Contains(t, err.Error(), "missing service name, caller name, encoding")
}

func TestInvalidStreamEmptyHeader(t *testing.T) {
	md := metadata.MD{
		CallerHeader:   []string{},
		ServiceHeader:  []string{"test"},
		EncodingHeader: []string{"raw"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), md)
	ss := &fakeServerStream{context: ctx}

	_, err := requestFromServerStream(ss, "service/proc")
	require.Contains(t, err.Error(), "code:invalid-argument")
	require.Contains(t, err.Error(), "missing caller name")
}

func TestInvalidStreamMultipleHeaders(t *testing.T) {
	md := metadata.MD{
		CallerHeader: []string{"caller1", "caller2"},
	}
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ss := &fakeServerStream{context: ctx}

	_, err := requestFromServerStream(ss, "service/proc")
	require.Contains(t, err.Error(), "code:invalid-argument")
	require.Contains(t, err.Error(), "header has more than one value: rpc-caller")
}
