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

// Package snappy provides a YARPC binding for snappy compression.
//
// To use snappy with gzip in particular, you will need to
// register the Compressor with grpc-go, the underlying implementation
// of gRPC which makes extensive use of global variables for
// depencencies.
//
//  import (
//      "google.golang.org/grpc/encoding"
//      "go.uber.org/yarpc/compressor/snappy"
//  )
//  encoding.RegisterCompressor(snappy.Compressor)
//
// If you are constructing your YARPC clients directly through the API,
// create a gRPC dialer with the Compressor option.
//
//  trans := grpc.NewTransport()
//  dialer := trans.NewDialer(grpc.Compressor(snappy.Compressor))
//  peers := roundrobin.New(dialer)
//  outbound := trans.NewOutbound(peers)
//
// If you are using the YARPC configurator to create YARPC objects
// using config files, you will also need to register the compressor
// with your configurator.
//
//  configurator := yarpcconfig.New()
//  configurator.MustRegisterCompressor(snappy.Compressor)
//
// Then, using the compression strategy for outbound requests
// on a particular client, just set the compressor to snappy.
//
//  outbounds:
//    theirsecureservice:
//      grpc:
//        address: ":443"
//        tls:
//          enabled: true
//        compressor: snappy
package snappy

import (
	"io"

	"github.com/golang/snappy"
	"go.uber.org/yarpc/api/transport"
)

// Compressor represents the snappy streaming compression strategy.
type Compressor struct{}

var _ transport.Compressor = Compressor{}

// Name is snappy.
func (Compressor) Name() string {
	return "snappy"
}

// Compress creates a snappy compressor.
func (Compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	return snappy.NewBufferedWriter(w), nil
}

// Decompress creates a snappy decompressor.
func (Compressor) Decompress(r io.Reader) (io.Reader, error) {
	return snappy.NewReader(r), nil
}
