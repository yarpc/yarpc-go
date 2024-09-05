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

package grpc

import (
	yarpcgrpccompressor "go.uber.org/yarpc/compressor/grpc"
	"go.uber.org/yarpc/yarpcconfig"
	"google.golang.org/grpc/encoding"
)

// Shared global state for tests, because sadly grpc-go uses global state for
// compressor registration.

var _metrics = newMetrics(nil, nil)

var (
	_goodCompressor  = newCompressor("test-good", testCompressorOk, _metrics)
	_badCompressor   = newCompressor("test-fail-comp", testCompressorFailToCompress, _metrics)
	_badDecompressor = newCompressor("test-fail-decomp", testCompressorFailToDecompress, _metrics)
	_gzipCompressor  = newCompressor("test-gzip", testCompressorGzip, _metrics)
)

var _configurator = yarpcconfig.New()

var _kit = _configurator.Kit("test")

func init() {
	compressors := []*testCompressor{
		_goodCompressor,
		_badCompressor,
		_badDecompressor,
		_gzipCompressor,
	}
	for _, compressor := range compressors {
		adapter := yarpcgrpccompressor.New(compressor)
		encoding.RegisterCompressor(adapter)
	}
}
