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

package server

import (
	"go.uber.org/yarpc/internal/crossdock/server/apachethrift"
	"go.uber.org/yarpc/internal/crossdock/server/googlegrpc"
	"go.uber.org/yarpc/internal/crossdock/server/http"
	"go.uber.org/yarpc/internal/crossdock/server/oneway"
	"go.uber.org/yarpc/internal/crossdock/server/tch"
	"go.uber.org/yarpc/internal/crossdock/server/yarpc"
)

// Start starts all required Crossdock test servers
func Start() {
	tch.Start()
	yarpc.Start()
	http.Start()
	apachethrift.Start()
	oneway.Start()
	googlegrpc.Start()
}

// Stop stops all required Crossdock test servers
func Stop() {
	tch.Stop()
	yarpc.Stop()
	http.Stop()
	apachethrift.Stop()
	oneway.Stop()
	googlegrpc.Stop()
}

// TODO(abg): We should probably use defers to ensure things that started up
// successfully are stopped before we exit.
