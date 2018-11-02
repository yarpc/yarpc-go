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

package internalgauntlettest

import (
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/v2"
)

// intended to be used with every request
func newCallOptions(resHeaders *map[string]string) []yarpc.CallOption {
	return []yarpc.CallOption{
		yarpc.ResponseHeaders(resHeaders),
		yarpc.WithHeader(_headerKeyReq, _headerValueReq),
		yarpc.WithShardKey(_shardKey),
		yarpc.WithRoutingKey(_routingKey),
		yarpc.WithRoutingDelegate(_routingDelegate),
	}
}

// intended to be invoked by an inbound handler
func validateCallOptions(call *yarpc.Call, encoding yarpc.Encoding) error {
	var errs error

	if _caller != call.Caller() {
		errs = multierr.Append(errs,
			fmt.Errorf("expected caller: %q, got: %q", _caller, call.Caller()))
	}
	if _service != call.Service() {
		errs = multierr.Append(errs,
			fmt.Errorf("expected service: %q, got: %q", _service, call.Service()))
	}
	if _routingKey != call.RoutingKey() {
		errs = multierr.Append(errs,
			fmt.Errorf("expected routingKey: %q, got: %q", _routingKey, call.RoutingKey()))
	}
	if _routingDelegate != call.RoutingDelegate() {
		errs = multierr.Append(errs,
			fmt.Errorf("expected routingDelegate: %q, got: %q", _routingDelegate, call.RoutingDelegate()))
	}
	if _shardKey != call.ShardKey() {
		errs = multierr.Append(errs,
			fmt.Errorf("expected shardKey: %q, got: %q", _shardKey, call.ShardKey()))
	}
	if encoding != call.Encoding() {
		errs = multierr.Append(errs,
			fmt.Errorf("expected encoding: %q, got: %q", encoding, call.Encoding()))
	}
	if _headerValueReq != call.Header(_headerKeyReq) {
		errs = multierr.Append(errs,
			fmt.Errorf("expected header: %q, got: %q", _headerValueReq, call.Header(_headerKeyReq)))
	}

	return errs
}
