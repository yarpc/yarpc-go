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
	"fmt"
	"time"

	"github.com/uber-go/mapdecode"
	"go.uber.org/multierr"
	iconfig "go.uber.org/yarpc/internal/config"
	"go.uber.org/yarpc/x/config"
)

// PolicySpec defines how to construct a retry Policy.
type PolicySpec struct {
	// Retries indicates the number of retries will be attempted on
	// failed requests.
	Retries uint `config:"retries"`

	// MaxRequestTimeout indicates the max timeout that every request
	// through the retry middleware will have.  If the timeout is greater
	// than the request timeout we'll use the request timeout instead.
	MaxRequestTimeout time.Duration `config:"maxtimeout"`

	// BackoffStrategy indicates which backoff strategy will be used between
	// each retry attempt.
	BackoffStrategy config.Backoff `config:"backoff"`
}

func (p PolicySpec) policy() (*Policy, error) {
	strategy, err := p.BackoffStrategy.Strategy()
	if err != nil {
		return nil, err
	}
	return NewPolicy(
		Retries(p.Retries),
		MaxRequestTimeout(p.MaxRequestTimeout),
		BackoffStrategy(strategy),
	), nil
}

// OverrideSpec defines per service or per service+procedure
// Policies that will be applied in the PolicyProvider.
type OverrideSpec struct {
	// Service is a YARPC service name for an override.
	Service string `config:"service"`

	// Procedure is a YARPC procedure name for an override.
	Procedure string `config:"procedure"`

	// WithPolicy specifies the policy name to use for the override.
	// It MUST reference an existing policy.
	WithPolicy string `config:"with"`
}

// PolicyProviderSpec is a definition of how to create a retry
// policy provider.
type PolicyProviderSpec struct {
	// NameToPolicies is a map of names to policy configs which
	// can be referenced later.
	NameToPolicies map[string]PolicySpec `config:"policies"`

	// Default is the Default policy that will be used.
	Default string `config:"default"`

	// Overrides are custom overrides for retry policies.
	Overrides []OverrideSpec `config:"overrides"`
}

// NewPolicyProvider creates a new policy provider that can be used in retry
// middleware
func NewPolicyProvider(src interface{}, opts ...mapdecode.Option) (*procedurePolicyProvider, error) {
	var err error
	var spec PolicyProviderSpec

	if err = iconfig.DecodeInto(&spec, src, opts...); err != nil {
		return nil, err
	}

	var nameToPolicy map[string]*Policy
	if nameToPolicy, err = spec.getPolicies(opts...); err != nil {
		return nil, err
	}

	return spec.getPolicyProvider(nameToPolicy)
}

func (spec PolicyProviderSpec) getPolicies(opts ...mapdecode.Option) (map[string]*Policy, error) {
	var errs error
	nameToPolicyMap := make(map[string]*Policy, len(spec.NameToPolicies))
	for name, policySpec := range spec.NameToPolicies {
		policy, err := policySpec.policy()
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		nameToPolicyMap[name] = policy
	}
	return nameToPolicyMap, errs
}

func (spec PolicyProviderSpec) getPolicyProvider(nameToPolicy map[string]*Policy) (*procedurePolicyProvider, error) {
	policyProvider := newProcedurePolicyProvider()

	var errs error
	if spec.Default != "" {
		if defaultPol, ok := nameToPolicy[spec.Default]; ok {
			policyProvider.registerDefault(defaultPol)
		} else {
			errs = multierr.Append(errs, fmt.Errorf("invalid default retry policy: %q, possiblities are: %v", spec.Default, keys(nameToPolicy)))
		}
	}

	for _, override := range spec.Overrides {
		pol, ok := nameToPolicy[override.WithPolicy]
		if !ok {
			errs = multierr.Append(errs, fmt.Errorf("invalid retry policy: %q, possiblities are: %v", override.WithPolicy, keys(nameToPolicy)))
			continue
		}

		if override.Service != "" && override.Procedure != "" {
			policyProvider.registerServiceProcedure(override.Service, override.Procedure, pol)
			continue
		}

		if override.Service != "" {
			policyProvider.registerService(override.Service, pol)
			continue
		}

		errs = multierr.Append(errs, fmt.Errorf("did not specify a service or procedure for retry policy override: %q", override.WithPolicy))
	}

	if errs != nil {
		return nil, errs
	}

	return policyProvider, nil
}

func keys(nameToPolicy map[string]*Policy) []string {
	ks := make([]string, 0, len(nameToPolicy))
	for k := range nameToPolicy {
		ks = append(ks, k)
	}
	return ks
}
