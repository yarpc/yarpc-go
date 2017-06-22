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

type serviceProcedure struct {
	Service   string
	Procedure string
}

// procedurePolicyProvider is a new PolicyProvider that
// has the ability to convert a context and transport request to
// determine which retry policy to use.
// The PolicyProvider has the ability to register policies based
// on service and procedure attributes.  It also has the ability
// to specify the default retry policy.
type procedurePolicyProvider struct {
	serviceProcedureToPolicy map[serviceProcedure]*Policy
	defaultPolicy            *Policy
}

func newProcedurePolicyProvider() *procedurePolicyProvider {
	defaultCopy := defaultPolicy
	return &procedurePolicyProvider{
		serviceProcedureToPolicy: make(map[serviceProcedure]*Policy),
		defaultPolicy:            &defaultCopy,
	}
}

func (ppp *procedurePolicyProvider) registerServiceProcedure(service, procedure string, pol *Policy) {
	ppp.serviceProcedureToPolicy[serviceProcedure{service, procedure}] = pol
}

func (ppp *procedurePolicyProvider) registerService(service string, pol *Policy) {
	ppp.serviceProcedureToPolicy[serviceProcedure{Service: service}] = pol
}

func (ppp *procedurePolicyProvider) registerDefault(pol *Policy) {
	ppp.defaultPolicy = pol
}

// GetPolicy returns a policy for the provided context and request.
func (ppp *procedurePolicyProvider) GetPolicy(_ context.Context, req *transport.Request) *Policy {
	if pol, ok := ppp.serviceProcedureToPolicy[serviceProcedure{req.Service, req.Procedure}]; ok {
		return pol
	}
	if pol, ok := ppp.serviceProcedureToPolicy[serviceProcedure{Service: req.Service}]; ok {
		return pol
	}
	return ppp.defaultPolicy
}
