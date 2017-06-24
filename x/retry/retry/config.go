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
	Retries           uint           `config:"retries"`
	MaxRequestTimeout time.Duration  `config:"maxtimeout"`
	BackoffStrategy   config.Backoff `config:"backoff"`
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

// OverrideSpec defines per Service or per service+procedure
// Policies that will be applied in the PolicyProvider.
type OverrideSpec struct {
	Service   string `config:"service"`
	Procedure string `config:"procedure"`
	With      string `config:"with"`
}

// PolicyProviderSpec is a definition of how to create a retry
// policy provider.
type PolicyProviderSpec struct {
	Policies  map[string]PolicySpec `config:"policies"`
	Default   string                `config:"default"`
	Overrides []OverrideSpec        `config:"overrides"`
}

// NewPolicyProvider creates a new policy provider that can be used in retry
// middleware
func NewPolicyProvider(src interface{}, opts ...mapdecode.Option) (*procedurePolicyProvider, error) {
	var err error
	var spec PolicyProviderSpec

	if err = iconfig.DecodeInto(&spec, src, opts...); err != nil {
		return nil, err
	}

	var policies map[string]*Policy
	if policies, err = spec.getPolicies(opts...); err != nil {
		return nil, err
	}

	return spec.getPolicyProvider(policies)
}

func (spec PolicyProviderSpec) getPolicies(opts ...mapdecode.Option) (map[string]*Policy, error) {
	var errs error
	policyMap := make(map[string]*Policy, len(spec.Policies))
	for name, policySpec := range spec.Policies {
		policy, err := policySpec.policy()
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		policyMap[name] = policy
	}
	return policyMap, errs
}

func (spec PolicyProviderSpec) getPolicyProvider(policies map[string]*Policy) (*procedurePolicyProvider, error) {
	policyProvider := newProcedurePolicyProvider()

	var errs error
	if spec.Default != "" {
		if defaultPol, ok := policies[spec.Default]; ok {
			policyProvider.registerDefault(defaultPol)
		} else {
			errs = multierr.Append(errs, fmt.Errorf("invalid default retry policy: %q, possiblities are: %v", spec.Default, keys(policies)))
		}
	}

	for _, override := range spec.Overrides {
		pol, ok := policies[override.With]
		if !ok {
			errs = multierr.Append(errs, fmt.Errorf("invalid retry policy: %q, possiblities are: %v", override.With, keys(policies)))
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

		errs = multierr.Append(errs, fmt.Errorf("did not specify a service or procedure for retry policy override: %q", override.With))
	}

	if errs != nil {
		return nil, errs
	}

	return policyProvider, nil
}

func keys(polMap map[string]*Policy) []string {
	ks := make([]string, 0, len(polMap))
	for k, _ := range polMap {
		ks = append(ks, k)
	}
	return ks
}
