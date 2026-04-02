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

// Package yarpczstd provides a YARPC binding for Zstandard compression.
package yarpczstd

import (
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
	"go.uber.org/yarpc/api/transport"
)

const _name = "zstd"

// Concurrency=1 means encoding and decoding happen in the calling goroutine
// with no background workers. With concurrency > 1 the zstd library spawns
// internal worker goroutines per encoder/decoder; because we pool those
// objects via sync.Pool, the workers would persist for the pool's lifetime.
const _defaultConcurrency = 1

// Option is an option argument for the Zstd compressor constructor, New.
type Option interface {
	apply(*Compressor)
}

// Level sets the compression level for the compressor.
// This is a shorthand for EncoderOptions(zstd.WithEncoderLevel(level)).
func Level(level zstd.EncoderLevel) Option {
	return encoderOptionsOption{opts: []zstd.EOption{zstd.WithEncoderLevel(level)}}
}

// EncoderOptions passes raw zstd encoder options through to the underlying
// encoder.
func EncoderOptions(opts ...zstd.EOption) Option {
	return encoderOptionsOption{opts: opts}
}

type encoderOptionsOption struct {
	opts []zstd.EOption
}

func (o encoderOptionsOption) apply(c *Compressor) {
	c.encoderOptions = append(c.encoderOptions, o.opts...)
}

// DecoderOptions passes raw zstd decoder options through to the underlying
// decoder.
func DecoderOptions(opts ...zstd.DOption) Option {
	return decoderOptionsOption{opts: opts}
}

type decoderOptionsOption struct {
	opts []zstd.DOption
}

func (o decoderOptionsOption) apply(c *Compressor) {
	c.decoderOptions = append(c.decoderOptions, o.opts...)
}

// New returns a Zstandard compression strategy, suitable for configuring
// an outbound dialer.
//
// The compressor needs to be adapted and registered to be compatible
// with the gRPC compressor system.
// Since gRPC requires global registration of compressors, you must arrange for
// the compressor to be registered in your application initialization.
// The adapter converts an io.Reader into an io.ReadCloser so that reading EOF
// will implicitly trigger Close, a behavior gRPC-go relies upon to reuse
// readers.
//
//	import (
//	    "google.golang.org/grpc/encoding"
//	    "go.uber.org/yarpc/compressor/grpc"
//	    "go.uber.org/yarpc/compressor/zstd"
//	)
//
//	var ZstdCompressor = yarpczstd.New(yarpczstd.Level(zstd.SpeedDefault))
//
//	func init() {
//	    zc := yarpcgrpccompressor.New(ZstdCompressor)
//	    encoding.RegisterCompressor(zc)
//	}
//
// If you are constructing your YARPC clients directly through the API,
// create a gRPC dialer with the Compressor option.
//
//	trans := grpc.NewTransport()
//	dialer := trans.NewDialer(grpc.Compressor(ZstdCompressor))
//	peers := roundrobin.New(dialer)
//	outbound := trans.NewOutbound(peers)
//
// If you are using the YARPC configurator to create YARPC objects
// using config files, you will also need to register the compressor
// with your configurator.
//
//	configurator := yarpcconfig.New()
//	configurator.MustRegisterCompressor(ZstdCompressor)
//
// Then, using the compression strategy for outbound requests
// on a particular client, just set the compressor to zstd.
//
//	outbounds:
//	  theirsecureservice:
//	    grpc:
//	      address: ":443"
//	      tls:
//	        enabled: true
//	      compressor: zstd
func New(opts ...Option) *Compressor {
	c := &Compressor{}
	for _, opt := range opts {
		opt.apply(c)
	}
	// Prepend defaults so user options win (last option wins in zstd).
	c.encoderOptions = append(
		[]zstd.EOption{zstd.WithEncoderConcurrency(_defaultConcurrency)},
		c.encoderOptions...,
	)
	c.decoderOptions = append(
		[]zstd.DOption{zstd.WithDecoderConcurrency(_defaultConcurrency)},
		c.decoderOptions...,
	)
	return c
}

// Compressor represents the zstd compression strategy.
type Compressor struct {
	encoderOptions []zstd.EOption
	decoderOptions []zstd.DOption
	encoders       sync.Pool
	decoders       sync.Pool
}

var _ transport.Compressor = (*Compressor)(nil)

// Name is zstd.
func (*Compressor) Name() string {
	return _name
}

// Compress creates a zstd compressor.
func (c *Compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	if cw, got := c.encoders.Get().(*writer); got {
		cw.encoder.Reset(w)
		return cw, nil
	}

	enc, err := zstd.NewWriter(w, c.encoderOptions...)
	if err != nil {
		return nil, err
	}

	return &writer{
		encoder: enc,
		pool:    &c.encoders,
	}, nil
}

type writer struct {
	encoder *zstd.Encoder
	pool    *sync.Pool
}

var _ io.WriteCloser = (*writer)(nil)

func (w *writer) Write(buf []byte) (int, error) {
	return w.encoder.Write(buf)
}

func (w *writer) Close() error {
	defer w.pool.Put(w)
	return w.encoder.Close()
}

// Decompress creates a zstd decompressor.
func (c *Compressor) Decompress(r io.Reader) (io.ReadCloser, error) {
	if dr, got := c.decoders.Get().(*reader); got {
		if err := dr.decoder.Reset(r); err != nil {
			c.decoders.Put(dr)
			return nil, err
		}
		return dr, nil
	}

	dec, err := zstd.NewReader(r, c.decoderOptions...)
	if err != nil {
		return nil, err
	}

	return &reader{
		decoder: dec,
		pool:    &c.decoders,
	}, nil
}

type reader struct {
	decoder *zstd.Decoder
	pool    *sync.Pool
}

var _ io.ReadCloser = (*reader)(nil)

func (r *reader) Read(buf []byte) (int, error) {
	return r.decoder.Read(buf)
}

func (r *reader) Close() error {
	r.pool.Put(r)
	return nil
}
