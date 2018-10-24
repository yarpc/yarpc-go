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

package yarpcrouter

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
)

var (
	_ yarpc.Router                = (*MapRouter)(nil)
	_ yarpc.UnaryTransportHandler = (*unaryTransportHandler)(nil)
)

type serviceProcedure struct {
	service   string
	procedure string
}

type serviceProcedureEncoding struct {
	service   string
	procedure string
	encoding  yarpc.Encoding
}

// MapRouter is a Router that maintains a map of the registered
// procedures.
type MapRouter struct {
	defaultService            string
	serviceProcedureEncodings map[serviceProcedureEncoding]yarpc.TransportProcedure
	serviceNames              map[string]struct{}
}

// NewMapRouter builds a new MapRouter that uses the given name as the
// default service name and registers the given procedures.
//
// If a provided procedure does not specify its service name, it will
// inherit the default service name. Multiple procedures with the
// same name and service name may exist if they handle different encodings.
// If a procedure does not specify an encoding, it can only support one handler.
// The router will select that handler regardless of the encoding.
func NewMapRouter(defaultService string, rs []yarpc.TransportProcedure) MapRouter {
	router := MapRouter{
		defaultService:            defaultService,
		serviceProcedureEncodings: make(map[serviceProcedureEncoding]yarpc.TransportProcedure),
		serviceNames:              map[string]struct{}{defaultService: {}},
	}

	router.register(rs)
	return router
}

// EncodingToTransportProcedures converts encoding-level procedures to transport-level procedures.
func EncodingToTransportProcedures(encodingProcedures []yarpc.EncodingProcedure) ([]yarpc.TransportProcedure, error) {
	transportProcedures := make([]yarpc.TransportProcedure, len(encodingProcedures))
	for i, p := range encodingProcedures {
		var transportHandlerSpec yarpc.TransportHandlerSpec
		switch p.HandlerSpec.Type() {
		case yarpc.Unary:
			transportHandlerSpec = yarpc.NewUnaryTransportHandlerSpec(&unaryTransportHandler{p})
		default:
			return nil, fmt.Errorf("unknown handler spec type: %v", p.HandlerSpec.Type())
		}

		transportProcedures[i] = yarpc.TransportProcedure{
			Name:        p.Name,
			Service:     p.Service,
			HandlerSpec: transportHandlerSpec,
			Encoding:    p.Encoding,
			Signature:   p.Signature,
		}
	}

	return transportProcedures, nil
}

// NewMapRouterWithProcedures constructs a new MapRouter with the given default service name and registers
// the given transport-level procedures.
func NewMapRouterWithProcedures(defaultService string, transportProcedures []yarpc.TransportProcedure) MapRouter {
	router := NewMapRouter(defaultService)
	router.Register(transportProcedures)
	return router
}

// Allows encoding-level procedures to act as transport-level procedures.
type unaryTransportHandler struct {
	h yarpc.EncodingProcedure
}

func (u *unaryTransportHandler) Handle(ctx context.Context, req *yarpc.Request, reqBuf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	decodedBody, err := u.h.Codec.Decode(reqBuf)
	if err != nil {
		return nil, nil, err
	}

	body, err := u.h.HandlerSpec.Unary().Handle(ctx, decodedBody)
	if err != nil {
		return nil, nil, err
	}

	encodedBody, err := u.h.Codec.Encode(body)
	if err != nil {
		return nil, nil, err
	}

	return nil, encodedBody, nil
}

// Register registers the procedure with the MapRouter.
// If the procedure does not specify its service name, the procedure will
// inherit the default service name of the router.
// Procedures should specify their encoding, and multiple procedures with the
// same name and service name can exist if they handle different encodings.
// If a procedure does not specify an encoding, it can only support one handler.
// The router will select that handler regardless of the encoding.
func (m MapRouter) register(rs []yarpc.TransportProcedure) {
	for _, r := range rs {
		if r.Service == "" {
			r.Service = m.defaultService
		}

		if r.Name == "" {
			panic("Expected procedure name not to be empty string in registration")
		}

		m.serviceNames[r.Service] = struct{}{}

		spe := serviceProcedureEncoding{
			service:   r.Service,
			procedure: r.Name,
			encoding:  r.Encoding,
		}

		// Route to individual handlers for unique combinations of service,
		// procedure, and encoding. This shall henceforth be the
		// recommended way for models to register procedures.
		m.serviceProcedureEncodings[spe] = r
	}
}

// Procedures returns a list procedures that
// have been registered so far.
func (m MapRouter) Procedures() []yarpc.TransportProcedure {
	procs := make([]yarpc.TransportProcedure, 0, len(m.serviceProcedureEncodings))
	for _, v := range m.serviceProcedureEncodings {
		procs = append(procs, v)
	}
	sort.Sort(sortableProcedures(procs))
	return procs
}

type sortableProcedures []yarpc.TransportProcedure

func (ps sortableProcedures) Len() int {
	return len(ps)
}

func (ps sortableProcedures) Less(i int, j int) bool {
	return ps[i].Less(ps[j])
}

func (ps sortableProcedures) Swap(i int, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

// Choose retrives the TransportHandlerSpec for the service, procedure, and encoding
// noted on the transport request, or returns an unrecognized procedure error
// (testable with yarpc.IsUnrecognizedProcedureError(err)).
func (m MapRouter) Choose(ctx context.Context, req *yarpc.Request) (yarpc.TransportHandlerSpec, error) {
	service, procedure, encoding := req.Service, req.Procedure, req.Encoding
	if service == "" {
		service = m.defaultService
	}

	if _, ok := m.serviceNames[service]; !ok {
		return yarpc.TransportHandlerSpec{},
			yarpcerror.Newf(yarpcerror.CodeUnimplemented, "unrecognized service name %q, "+
				"available services: %s", req.Service, getAvailableServiceNames(m.serviceNames))
	}

	// Fully specified combinations of service, procedure, and encoding.
	spe := serviceProcedureEncoding{
		service:   service,
		procedure: procedure,
		encoding:  encoding,
	}
	if procedure, ok := m.serviceProcedureEncodings[spe]; ok {
		return procedure.HandlerSpec, nil
	}

	return yarpc.TransportHandlerSpec{}, yarpcerror.Newf(yarpcerror.CodeUnimplemented, "unrecognized procedure %q for service %q", req.Procedure, req.Service)
}

// Extract keys from service names map and return a formatted string
func getAvailableServiceNames(svcMap map[string]struct{}) string {
	var serviceNames []string
	for key := range svcMap {
		serviceNames = append(serviceNames, strconv.Quote(key))
	}
	// Sort the string array to generate consistent result
	sort.Strings(serviceNames)
	return strings.Join(serviceNames, ", ")
}
