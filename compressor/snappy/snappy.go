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

// Package yarpcsnappy provides a YARPC binding for snappy compression.
package yarpcsnappy

import (
	"io"
	"sync"

	"github.com/golang/snappy"
	"go.uber.org/yarpc/api/transport"
)

// Option is an option argument for the Snappy compressor constructor, New.
type Option interface {
	apply(*Compressor)
}

// New returns a Snappy compression strategy, suitable for configuring
// an outbound dialer.
//
// The compressor is compatible with the gRPC experimental compressor system.
// However, since gRPC requires global registration of compressors,
// you must arrange for the compressor to be registered in your
// application initialization.
//
//	import (
//	    "google.golang.org/grpc/encoding"
//	    "go.uber.org/yarpc/compressor/grpc"
//	    "go.uber.org/yarpc/compressor/snappy"
//	)
//
//	var SnappyCompressor = yarpcsnappy.New()
//
//	func init()
//	    sc := yarpcgrpccompressor.New(SnappyCompressor)
//	    encoding.RegisterCompressor(sc)
//	}
//
// If you are constructing your YARPC clients directly through the API,
// create a gRPC dialer with the Compressor option.
//
//	trans := grpc.NewTransport()
//	dialer := trans.NewDialer(grpc.Compressor(SnappyCompressor))
//	peers := roundrobin.New(dialer)
//	outbound := trans.NewOutbound(peers)
//
// If you are using the YARPC configurator to create YARPC objects
// using config files, you will also need to register the compressor
// with your configurator.
//
//	configurator := yarpcconfig.New()
//	configurator.MustRegisterCompressor(SnappyCompressor)
//
// Then, using the compression strategy for outbound requests
// on a particular client, just set the compressor to snappy.
//
//	outbounds:
//	  theirsecureservice:
//	    grpc:
//	      address: ":443"
//	      tls:
//	        enabled: true
//	      compressor: snappy
func New(...Option) *Compressor {
	return &Compressor{}
}

// Compressor represents the snappy streaming compression strategy.
type Compressor struct {
	compressors   sync.Pool
	decompressors sync.Pool
}

var _ transport.Compressor = (*Compressor)(nil)

// Name is snappy.
func (*Compressor) Name() string {
	return "snappy"
}

// Compress creates a snappy compressor.
func (c *Compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	if cw, got := c.compressors.Get().(*writer); got {
		cw.writer.Reset(w)
		return cw, nil
	}
	return &writer{
		writer: snappy.NewBufferedWriter(w),
		pool:   &c.compressors,
	}, nil
}

type writer struct {
	writer *snappy.Writer
	pool   *sync.Pool
}

var _ io.WriteCloser = (*writer)(nil)

func (w *writer) Write(buf []byte) (int, error) {
	return w.writer.Write(buf)
}

func (w *writer) Close() error {
	defer w.pool.Put(w)
	return w.writer.Close()
}

// Decompress creates a snappy decompressor.
func (c *Compressor) Decompress(r io.Reader) (io.ReadCloser, error) {
	dr, got := c.decompressors.Get().(*reader)
	if got {
		dr.reader.Reset(r)
		return dr, nil
	}
	return &reader{
		reader: snappy.NewReader(r),
		pool:   &c.decompressors,
	}, nil
}

type reader struct {
	reader *snappy.Reader
	pool   *sync.Pool
}

var _ io.ReadCloser = (*reader)(nil)

func (r *reader) Read(buf []byte) (n int, err error) {
	return r.reader.Read(buf)
}

func (r *reader) Close() error {
	r.pool.Put(r)
	return nil
}
