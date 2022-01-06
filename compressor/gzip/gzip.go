// Copyright (c) 2022 Uber Technologies, Inc.
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

// Package yarpcgzip provides a YARPC binding for GZIP compression.
package yarpcgzip

import (
	"compress/gzip"
	"encoding/binary"
	"io"
	"sync"

	"go.uber.org/yarpc/api/transport"
)

const name = "gzip"

// Option is an option argument for the Gzip compressor constructor, New.
type Option interface {
	apply(*Compressor)
}

// Level sets the compression level for the compressor.
func Level(level int) Option {
	return levelOption{level: level}
}

type levelOption struct {
	level int
}

func (o levelOption) apply(opts *Compressor) {
	opts.level = o.level
}

// New returns a GZIP compression strategy, suitable for configuring an
// outbound dialer.
//
// The compressor needs to be adapted and registered to be compatible
// with the gRPC compressor system.
// Since gRPC requires global registration of compressors, you must arrange for
// the compressor to be registered in your application initialization.
// The adapter converts an io.Reader into an io.ReadCloser so that reading EOF
// will implicitly trigger Close, a behavior gRPC-go relies upon to reuse
// readers.
//
//  import (
//      "compress/gzip"
//
//      "google.golang.org/grpc/encoding"
//      "go.uber.org/yarpc/compressor/grpc"
//      "go.uber.org/yarpc/compressor/gzip"
//  )
//
//  var GZIPCompressor = yarpcgzip.New(yarpcgzip.Level(gzip.BestCompression))
//
//  func init()
//      gz := yarpcgrpccompressor.New(GZIPCompressor)
//      encoding.RegisterCompressor(gz)
//  }
//
// If you are constructing your YARPC clients directly through the API,
// create a gRPC dialer with the Compressor option.
//
//  trans := grpc.NewTransport()
//  dialer := trans.NewDialer(GZIPCompressor)
//  peers := roundrobin.New(dialer)
//  outbound := trans.NewOutbound(peers)
//
// If you are using the YARPC configurator to create YARPC objects
// using config files, you will also need to register the compressor
// with your configurator.
//
//  configurator := yarpcconfig.New()
//  configurator.MustRegisterCompressor(GZIPCompressor)
//
// Then, using the compression strategy for outbound requests
// on a particular client, just set the compressor to gzip.
//
//  outbounds:
//    theirsecureservice:
//      grpc:
//        address: ":443"
//        tls:
//          enabled: true
//        compressor: gzip
//
func New(opts ...Option) *Compressor {
	c := &Compressor{
		level: gzip.DefaultCompression,
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

// Compressor represents the gzip compression strategy.
type Compressor struct {
	level         int
	compressors   sync.Pool
	decompressors sync.Pool
}

var _ transport.Compressor = (*Compressor)(nil)

// Name is gzip.
func (*Compressor) Name() string {
	return name
}

// Compress creates a gzip compressor.
func (c *Compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	if cw, got := c.compressors.Get().(*writer); got {
		cw.writer.Reset(w)
		return cw, nil
	}

	cw, err := gzip.NewWriterLevel(w, c.level)
	if err != nil {
		return nil, err
	}

	return &writer{
		writer: cw,
		pool:   &c.compressors,
	}, nil
}

type writer struct {
	writer *gzip.Writer
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

// Decompress obtains a gzip decompressor.
func (c *Compressor) Decompress(r io.Reader) (io.ReadCloser, error) {
	if dr, got := c.decompressors.Get().(*reader); got {
		if err := dr.reader.Reset(r); err != nil {
			c.decompressors.Put(r)
			return nil, err
		}

		return dr, nil
	}

	dr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &reader{
		reader: dr,
		pool:   &c.decompressors,
	}, nil
}

type reader struct {
	reader *gzip.Reader
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

// DecompressedSize returns the decompressed size of the given GZIP compressed
// bytes.
//
// gRPC specifically casts the compressor to a DecompressedSizer
// to pre-check message length.
//
// Per gRPC-go, on which this is based:
// https://github.com/grpc/grpc-go/blob/master/encoding/gzip/gzip.go
//
// RFC1952 specifies that the last four bytes "contains the size of
// the original (uncompressed) input data modulo 2^32."
// gRPC has a max message size of 2GB so we don't need to worry about
// wraparound for that transport protocol.
func (c *Compressor) DecompressedSize(buf []byte) int {
	last := len(buf)
	if last < 4 {
		return -1
	}
	return int(binary.LittleEndian.Uint32(buf[last-4 : last]))
}
