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

// Package yarpcgrpccompressor provides an adapter for YARPC compressors to
// gRPC compressors.
//
// The only distinction is that YARPC's Decompressor returns an io.ReadCloser
// instead of a mere io.Reader.
// gRPC does not call Close, so must infer the end of stream by returning the
// reader to the reader pool when Read returns io.EOF.
// This wrapper uses the io.EOF to trigger and automatic Close.
package yarpcgrpccompressor

import (
	"io"

	"go.uber.org/yarpc/api/transport"
	"google.golang.org/grpc/encoding"
)

// New adapts a YARPC compressor to a gRPC compressor.
func New(compressor transport.Compressor) encoding.Compressor {
	return &Compressor{compressor: compressor}
}

// Compressor is a gRPC compressor that wraps a YARPC compressor.
type Compressor struct {
	compressor transport.Compressor
}

var _ encoding.Compressor = (*Compressor)(nil)

// Name returns the name of the underlying compressor.
func (c *Compressor) Name() string {
	return c.compressor.Name()
}

// Compress wraps a writer with a compressing writer.
func (c *Compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	return c.compressor.Compress(w)
}

// Decompress wraps a reader with a decompressing reader.
func (c *Compressor) Decompress(r io.Reader) (io.Reader, error) {
	dr, err := c.compressor.Decompress(r)
	if err != nil {
		return nil, err
	}
	return &reader{reader: dr}, nil
}

type reader struct {
	reader io.ReadCloser
}

var _ io.Reader = (*reader)(nil)

func (r *reader) Read(buf []byte) (n int, err error) {
	n, err = r.reader.Read(buf)
	if err == io.EOF {
		r.reader.Close()
	}
	return n, err
}
