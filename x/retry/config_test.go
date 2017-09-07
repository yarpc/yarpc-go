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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/internal/whitespace"
	"gopkg.in/yaml.v2"
)

func TestConfig(t *testing.T) {
	type testStruct struct {
		msg string

		retryConfig string

		wantPolicyProvider *ProcedurePolicyProvider
		wantError          []string
	}
	tests := []testStruct{
		{
			msg: "just default policy",
			retryConfig: `
				policies:
					once:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: once
			`,
			wantPolicyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(time.Millisecond*500),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*10),
							backoff.MaxBackoff(time.Second*1),
						)),
					),
				),
			).provider,
		},
		{
			msg: "service+default policy",
			retryConfig: `
				policies:
					once:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
					service:
						retries: 5
						maxtimeout: 100ms
						backoff:
							exponential:
								first: 100ms
								max: 10s
				default: once
				overrides:
					- service: myservice
					  with: service
			`,
			wantPolicyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(time.Millisecond*500),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*10),
							backoff.MaxBackoff(time.Second*1),
						)),
					),
				),
			).registerService(
				"myservice",
				NewPolicy(
					Retries(5),
					MaxRequestTimeout(time.Millisecond*100),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*100),
							backoff.MaxBackoff(time.Second*10),
						)),
					),
				),
			).provider,
		},
		{
			msg: "procedure+default policy",
			retryConfig: `
				policies:
					once:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
					procedure:
						retries: 5
						maxtimeout: 100ms
						backoff:
							exponential:
								first: 100ms
								max: 10s
				default: once
				overrides:
					- procedure: myproc
					  with: procedure
			`,
			wantPolicyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(time.Millisecond*500),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*10),
							backoff.MaxBackoff(time.Second*1),
						)),
					),
				),
			).registerProcedure(
				"myproc",
				NewPolicy(
					Retries(5),
					MaxRequestTimeout(time.Millisecond*100),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*100),
							backoff.MaxBackoff(time.Second*10),
						)),
					),
				),
			).provider,
		},
		{
			msg: "service+serviceproc+default policy",
			retryConfig: `
				policies:
					once:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
					service:
						retries: 5
						maxtimeout: 100ms
						backoff:
							exponential:
								first: 100ms
								max: 10s
					serviceproc:
						retries: 10
						maxtimeout: 20ms
						backoff:
							exponential:
								first: 20ms
								max: 5s
				default: once
				overrides:
					- service: myservice
					  with: service
					- service: myservice
					  procedure: myprocedure
					  with: serviceproc
			`,
			wantPolicyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(time.Millisecond*500),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*10),
							backoff.MaxBackoff(time.Second*1),
						)),
					),
				),
			).registerService(
				"myservice",
				NewPolicy(
					Retries(5),
					MaxRequestTimeout(time.Millisecond*100),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*100),
							backoff.MaxBackoff(time.Second*10),
						)),
					),
				),
			).registerServiceProcedure(
				"myservice",
				"myprocedure",
				NewPolicy(
					Retries(10),
					MaxRequestTimeout(time.Millisecond*20),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*20),
							backoff.MaxBackoff(time.Second*5),
						)),
					),
				),
			).provider,
		},
		{
			msg: "unused policy",
			retryConfig: `
				policies:
					once:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
					unused:
						retries: 5
						maxtimeout: 100ms
						backoff:
							exponential:
								first: 100ms
								max: 10s
				default: once
			`,
			wantPolicyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(time.Millisecond*500),
					BackoffStrategy(
						exponentialNoError(backoff.NewExponential(
							backoff.FirstBackoff(time.Millisecond*10),
							backoff.MaxBackoff(time.Second*1),
						)),
					),
				),
			).provider,
		},
		{
			msg: "invalid default policy",
			retryConfig: `
				policies:
					realpolicy:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: fakepolicy
			`,
			wantError: []string{
				`invalid default retry policy: "fakepolicy", possibilities are:`,
				`realpolicy`,
			},
		},
		{
			msg: "invalid override policy",
			retryConfig: `
				policies:
					realpolicy:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: realpolicy
				overrides:
					- service: myservice
					  with: fakepolicy
			`,
			wantError: []string{
				`invalid retry policy: "fakepolicy", possibilities are:`,
				`realpolicy`,
			},
		},
		{
			msg: "missing with in override policy",
			retryConfig: `
				policies:
					realpolicy:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: realpolicy
				overrides:
					- service: myservice
			`,
			wantError: []string{
				`invalid retry policy: "", possibilities are:`,
				`realpolicy`,
			},
		},
		{
			msg: "no service or procedure in override policy",
			retryConfig: `
				policies:
					realpolicy:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: realpolicy
				overrides:
					- with: realpolicy
			`,
			wantError: []string{
				`did not specify a service or procedure for retry policy override: "realpolicy"`,
			},
		},
		{
			msg: "invalid policy retries",
			retryConfig: `
				policies:
					realpolicy:
						retries: -1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: realpolicy
			`,
			wantError: []string{
				`cannot parse`,
				`realpolicy`,
			},
		},
		{
			msg: "invalid policy timeout",
			retryConfig: `
				policies:
					realpolicy:
						retries: 1
						maxtimeout: abc
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: realpolicy
			`,
			wantError: []string{
				`error decoding`,
				`realpolicy`,
			},
		},
		{
			msg: "invalid policy backoff",
			retryConfig: `
				policies:
					realpolicy:
						retries: 1
						maxtimeout: abc
						backoff: error
				default: realpolicy
			`,
			wantError: []string{
				`error decoding`,
				`realpolicy`,
			},
		},
		{
			msg:                "fallsback to default policy",
			retryConfig:        "",
			wantPolicyProvider: newPolicyProviderBuilder().provider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			var data map[string]interface{}
			err := yaml.Unmarshal([]byte(whitespace.Expand(tt.retryConfig)), &data)
			require.NoError(t, err, "error unmarshalling")

			middleware, stopFunc, err := NewUnaryMiddlewareFromConfig(data)
			defer stopFunc()
			if len(tt.wantError) > 0 {
				require.Error(t, err, "expected error, got none")
				for _, wantErr := range tt.wantError {
					require.Contains(t, err.Error(), wantErr, "expected error")
				}
				return
			}
			require.NoError(t, err, "error decoding")

			policyProvider, ok := middleware.provider.(*ProcedurePolicyProvider)
			require.True(t, ok, "PolicyProvider was not a ProcedurePolicyProvider")
			assertPoliciesAreEqual(t, tt.wantPolicyProvider.defaultPolicy, policyProvider.defaultPolicy)

			assert.Equal(t, len(tt.wantPolicyProvider.serviceProcedureToPolicy), len(policyProvider.serviceProcedureToPolicy), "mismatch in number of retry policies")
			for sp, expectedPolicy := range tt.wantPolicyProvider.serviceProcedureToPolicy {
				actualPolicy, ok := policyProvider.serviceProcedureToPolicy[sp]
				if !assert.True(t, ok, "missing mapping for serviceprocedure: %v", sp) {
					continue
				}
				assertPoliciesAreEqual(t, expectedPolicy, actualPolicy)
			}
		})
	}
}

func exponentialNoError(exp *backoff.ExponentialStrategy, _ error) *backoff.ExponentialStrategy {
	return exp
}

func assertPoliciesAreEqual(t *testing.T, expected, actual *Policy) {
	if expected == nil {
		assert.Nil(t, actual)
		return
	}
	require.NotNil(t, expected)
	assert.Equal(t, expected.opts.retries, actual.opts.retries, "did not match on retries")
	assert.Equal(t, expected.opts.maxRequestTimeout, actual.opts.maxRequestTimeout, "did not match on maxRequestTimeout")

	expectedStrat, ok := expected.opts.backoffStrategy.(*backoff.ExponentialStrategy)
	require.True(t, ok, "first strategy was not an exponential strategy")

	actualStrat, ok := actual.opts.backoffStrategy.(*backoff.ExponentialStrategy)
	require.True(t, ok, "second strategy was not an exponential strategy")

	assert.True(
		t,
		expectedStrat.IsEqual(actualStrat),
		fmt.Sprintf("expected backoff %v is not equalt to actual backoff %v", expectedStrat, actualStrat),
	)
}
