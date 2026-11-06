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

package thrift

import (
	"bufio"
	"context"
	"io"

	"go.uber.org/multierr"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/wire"
	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/observabilitylogger"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/zap"
)

var _emptyResponse = NoWireResponse{}
var _loggedMissingAppErrMetaSetter bool

// NoWireCall contains all of the required objects needed for an underlying
// Handle needs to unpack any given request.
//
// Reader may be a buffered view of the request body. It is only valid for the
// duration of the HandleNoWire call and must not be retained or read from
// another goroutine after HandleNoWire returns.
type NoWireCall struct {
	Reader        io.Reader
	RequestReader stream.RequestReader
	EnvelopeType  wire.EnvelopeType
}

// NoWireHandler is implemented by the generated code for each method that the
// server needs to implement. It is responsible for unpacking the request,
// executing it, and returning a NoWireResponse that contains information about
// how to construct a response as well as any relevant metadata while executing
// the request.
type NoWireHandler interface {
	HandleNoWire(context.Context, *NoWireCall) (NoWireResponse, error)
}

// thriftNoWireHandler is similar to thriftUnaryHandler and thriftOnewayHandler
// except that thriftNoWireHandler implements both transport.UnaryHandler and
// transport.OnewayHandler through a single type, utilizing the "nowire"
// (streaming in the ThriftRW context, bypassing the intermediate "Wire"
// representation) implementation.
//
// The UnaryHandler and OnewayHandler implementations then call into type that
// implements NoWireHandler, which understands how to unpack and invoke the
// request, given the ThriftRW primitives for reading the raw representation.
type thriftNoWireHandler struct {
	Handler       NoWireHandler
	RequestReader stream.RequestReader
}

var (
	_ transport.OnewayHandler = (*thriftNoWireHandler)(nil)
	_ transport.UnaryHandler  = (*thriftNoWireHandler)(nil)
)

func (t thriftNoWireHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) (err error) {
	ctx, call := encodingapi.NewInboundCall(ctx)
	defer func() {
		err = multierr.Append(err, closeReader(treq.Body))
	}()

	res, err := t.decodeAndHandle(ctx, call, treq, rw, wire.Call)
	if err != nil {
		return err
	}

	if resType := res.Body.EnvelopeType(); resType != wire.Reply {
		return errors.ResponseBodyEncodeError(
			treq, errUnexpectedEnvelopeType(resType))
	}

	if res.IsApplicationError {
		rw.SetApplicationError()
		if applicationErrorMetaSetter, ok := rw.(transport.ApplicationErrorMetaSetter); ok {
			applicationErrorMetaSetter.SetApplicationErrorMeta(&transport.ApplicationErrorMeta{
				Details: res.ApplicationErrorDetails,
				Name:    res.ApplicationErrorName,
				Code:    res.ApplicationErrorCode,
			})
		} else if logger := observabilitylogger.FromContext(ctx); logger != nil {
			if !_loggedMissingAppErrMetaSetter {
				_loggedMissingAppErrMetaSetter = true
				logger.Warn("ResponseWriter does not implement ApplicationErrorMetaSetter; application error metadata will not be propagated to transport",
					zap.Stack("stacktrace"),
				)
			}
		}
	}

	if err := call.WriteToResponse(rw); err != nil {
		// not reachable
		return err
	}

	if err := res.ResponseWriter.WriteResponse(wire.Reply, rw, res.Body); err != nil {
		return errors.ResponseBodyEncodeError(treq, err)
	}

	return nil
}

func (t thriftNoWireHandler) HandleOneway(ctx context.Context, treq *transport.Request) (err error) {
	ctx, call := encodingapi.NewInboundCall(ctx)
	defer func() {
		err = multierr.Append(err, closeReader(treq.Body))
	}()

	_, err = t.decodeAndHandle(ctx, call, treq, nil, wire.OneWay)
	return err
}

// decodeAndHandle is a shared utility between the implementations of
// transport.UnaryHandler and transport.OnewayHandler, to decode and execute
// the request regardless of enveloping, via the "nowire" implementation in
// ThriftRW.
func (t thriftNoWireHandler) decodeAndHandle(
	ctx context.Context,
	call *encodingapi.InboundCall,
	treq *transport.Request,
	rw transport.ResponseWriter,
	reqEnvelopeType wire.EnvelopeType,
) (NoWireResponse, error) {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return _emptyResponse, err
	}

	if err := call.ReadFromRequest(treq); err != nil {
		return _emptyResponse, err
	}

	// The buffered reader is only valid while the handler runs. Generated
	// handlers decode the whole request inside HandleNoWire; the transport
	// still owns and closes treq.Body afterwards.
	reader, release := bufferNoWireReader(treq.Body)
	defer release()
	if br, ok := reader.(*bufio.Reader); ok {
		// ThriftRW's Protocol.ReadRequest sniffs for an envelope with a single
		// Read of two bytes, and treats a shorter read as an empty struct. A
		// transport body may legitimately return one byte first (for example
		// when only the first HTTP/2 DATA chunk has arrived), which would
		// misdecode the request. Fill the buffer so that read always succeeds.
		// An error here is not lost: the next Read returns it after any
		// buffered bytes.
		_, _ = br.Peek(2)
	}

	nwc := NoWireCall{
		Reader:        reader,
		EnvelopeType:  reqEnvelopeType,
		RequestReader: t.RequestReader,
	}

	return t.Handler.HandleNoWire(ctx, &nwc)
}
