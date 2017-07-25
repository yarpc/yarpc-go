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

	"go.uber.org/multierr"
	iconfig "go.uber.org/yarpc/internal/config"
	"go.uber.org/yarpc/x/config"
)

// PolicyConfig defines how to construct a retry Policy.
type PolicyConfig struct {
	// Retries indicates the number of retries that will be attempted on failed
	// requests.
	Retries uint `config:"retries"`

	// MaxTimeout indicates the max timeout that every request through the retry
	// middleware will have.  If the timeout is greater than the request timeout
	// we'll use the request timeout instead.
	MaxTimeout time.Duration `config:"maxTimeout"`

	// BackoffStrategy defines a backoff strategy in place by embedding a
	// backoff config.
	BackoffStrategy config.Backoff `config:"backoff"`
}

func (p PolicyConfig) policy() (*Policy, error) {
	strategy, err := p.BackoffStrategy.Strategy()
	if err != nil {
		return nil, err
	}
	return NewPolicy(
		Retries(p.Retries),
		MaxRequestTimeout(p.MaxTimeout),
		BackoffStrategy(strategy),
	), nil
}

// PolicyOverrideConfig defines per service or per service+procedure Policies
// that will be applied in the PolicyProvider.
type PolicyOverrideConfig struct {
	// Service is a YARPC service name for an override.
	Service string `config:"service"`

	// Procedure is a YARPC procedure name for an override.
	Procedure string `config:"procedure"`

	// WithPolicy specifies the policy name to use for the override. It MUST
	// reference an existing policy.
	WithPolicy string `config:"with"`
}

// MiddlewareConfig is a definition of how to create a retry middleware.
type MiddlewareConfig struct {
	// NameToPolicies is a map of names to policy configs which can be
	// referenced later.
	NameToPolicies map[string]PolicyConfig `config:"policies"`

	// Default is the name of the default policy that will be used.
	Default string `config:"default"`

	// PolicyOverrides allow changing the retry policies for requests matching
	// certain criteria.
	PolicyOverrides []PolicyOverrideConfig `config:"overrides"`
}

// NewUnaryMiddlewareFromConfig creates a new policy provider that can be used
// in retry middleware.
func NewUnaryMiddlewareFromConfig(src interface{}, opts ...MiddlewareOption) (*OutboundMiddleware, error) {
	var cfg MiddlewareConfig
	if err := iconfig.DecodeInto(&cfg, src); err != nil {
		return nil, err
	}

	nameToPolicy, err := cfg.getPolicies()
	if err != nil {
		return nil, err
	}

	policyProvider, err := cfg.getPolicyProvider(nameToPolicy)
	if err != nil {
		return nil, err
	}

	opts = append(opts, WithPolicyProvider(policyProvider))
	return NewUnaryMiddleware(opts...), nil
}

func (cfg MiddlewareConfig) getPolicies() (map[string]*Policy, error) {
	var errs error
	nameToPolicyMap := make(map[string]*Policy, len(cfg.NameToPolicies))
	for name, policyConfig := range cfg.NameToPolicies {
		policy, err := policyConfig.policy()
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		nameToPolicyMap[name] = policy
	}
	return nameToPolicyMap, errs
}

func (cfg MiddlewareConfig) getPolicyProvider(nameToPolicy map[string]*Policy) (*ProcedurePolicyProvider, error) {
	policyProvider := NewProcedurePolicyProvider()

	var errs error
	if cfg.Default != "" {
		if defaultPol, ok := nameToPolicy[cfg.Default]; ok {
			policyProvider.SetDefault(defaultPol)
		} else {
			errs = multierr.Append(errs, fmt.Errorf("invalid default retry policy: %q, possibilities are: %v", cfg.Default, policyNames(nameToPolicy)))
		}
	}

	for _, override := range cfg.PolicyOverrides {
		pol, ok := nameToPolicy[override.WithPolicy]
		if !ok {
			errs = multierr.Append(errs, fmt.Errorf("invalid retry policy: %q, possibilities are: %v", override.WithPolicy, policyNames(nameToPolicy)))
			continue
		}

		if override.Service != "" && override.Procedure != "" {
			policyProvider.RegisterServiceProcedure(override.Service, override.Procedure, pol)
			continue
		}

		if override.Service != "" {
			policyProvider.RegisterService(override.Service, pol)
			continue
		}

		if override.Procedure != "" {
			policyProvider.RegisterProcedure(override.Procedure, pol)
			continue
		}

		errs = multierr.Append(errs, fmt.Errorf("did not specify a service or procedure for retry policy override: %q", override.WithPolicy))
	}

	return policyProvider, errs
}

func policyNames(nameToPolicy map[string]*Policy) []string {
	ks := make([]string, 0, len(nameToPolicy))
	for k := range nameToPolicy {
		ks = append(ks, k)
	}
	return ks
}
