// Copyright (c) 2017 Uber Technologies, Inc.
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

package retry

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// PolicyProvider returns a retry policy to use for the given context and
// request.  Nil responses will be interpreted as "no retries".
type PolicyProvider interface {
	Policy(context.Context, *transport.Request) *Policy
}

type serviceProcedure struct {
	Service   string
	Procedure string
}

// ProcedurePolicyProvider is a PolicyProvider that has the ability to resolve
// retry policies based on the Service and Procedure of each retry request. It
// also supports returning a default policy if no Service or Procedure matched.
type ProcedurePolicyProvider struct {
	serviceProcedureToPolicy map[serviceProcedure]*Policy
	defaultPolicy            *Policy
}

// NewProcedurePolicyProvider creates a new ProcedurePolicyProvider.
func NewProcedurePolicyProvider() *ProcedurePolicyProvider {
	return &ProcedurePolicyProvider{
		serviceProcedureToPolicy: make(map[serviceProcedure]*Policy),
		defaultPolicy:            nil,
	}
}

// RegisterServiceProcedure registers a new Policy that will be used for retries
// if there is a Service+Procedure match on the request.
func (ppp *ProcedurePolicyProvider) RegisterServiceProcedure(service, procedure string, pol *Policy) {
	ppp.serviceProcedureToPolicy[serviceProcedure{Service: service, Procedure: procedure}] = pol
}

// RegisterService registers a new Policy that will be used for retries if there
// is a Service match on the request.
func (ppp *ProcedurePolicyProvider) RegisterService(service string, pol *Policy) {
	ppp.serviceProcedureToPolicy[serviceProcedure{Service: service}] = pol
}

// RegisterDefault registers a default retry Policy that will be used if there
// are no matches for any other policy (based on Service or Procedure).
func (ppp *ProcedurePolicyProvider) RegisterDefault(pol *Policy) {
	ppp.defaultPolicy = pol
}

// Policy returns a policy for the provided context and request.
func (ppp *ProcedurePolicyProvider) Policy(_ context.Context, req *transport.Request) *Policy {
	if pol, ok := ppp.serviceProcedureToPolicy[serviceProcedure{req.Service, req.Procedure}]; ok {
		return pol
	}
	if pol, ok := ppp.serviceProcedureToPolicy[serviceProcedure{Service: req.Service}]; ok {
		return pol
	}
	return ppp.defaultPolicy
}
