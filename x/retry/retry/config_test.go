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
	"testing"
	"time"

	"github.com/crossdock/crossdock-go/assert"
	"github.com/crossdock/crossdock-go/require"
	"go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/internal/whitespace"
	"gopkg.in/yaml.v2"
)

func TestConfig(t *testing.T) {
	type testStruct struct {
		msg string

		retryConfig string

		wantPolicyProvider *procedurePolicyProvider
		wantError          []string
	}
	tests := []testStruct{
		{
			msg: "just default policy",
			retryConfig: whitespace.Expand(`
				policies:
					once:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: once
			`),
			wantPolicyProvider: newPolicyProviderBuilder().registerDefault(
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
			retryConfig: whitespace.Expand(`
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
			`),
			wantPolicyProvider: newPolicyProviderBuilder().registerDefault(
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
			msg: "service+serviceproc+default policy",
			retryConfig: whitespace.Expand(`
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
			`),
			wantPolicyProvider: newPolicyProviderBuilder().registerDefault(
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
			retryConfig: whitespace.Expand(`
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
			`),
			wantPolicyProvider: newPolicyProviderBuilder().registerDefault(
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
			retryConfig: whitespace.Expand(`
				policies:
					realpolicy:
						retries: 1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: fakepolicy
			`),
			wantError: []string{
				`invalid default retry policy: "fakepolicy", possiblities are:`,
				`realpolicy`,
			},
		},
		{
			msg: "invalid override policy",
			retryConfig: whitespace.Expand(`
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
			`),
			wantError: []string{
				`invalid retry policy: "fakepolicy", possiblities are:`,
				`realpolicy`,
			},
		},
		{
			msg: "missing with in override policy",
			retryConfig: whitespace.Expand(`
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
			`),
			wantError: []string{
				`invalid retry policy: "", possiblities are:`,
				`realpolicy`,
			},
		},
		{
			msg: "no service or procedure in override policy",
			retryConfig: whitespace.Expand(`
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
			`),
			wantError: []string{
				`did not specify a service or procedure for retry policy override: "realpolicy"`,
			},
		},
		{
			msg: "invalid policy retries",
			retryConfig: whitespace.Expand(`
				policies:
					realpolicy:
						retries: -1
						maxtimeout: 500ms
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: realpolicy
			`),
			wantError: []string{
				`cannot parse`,
				`realpolicy`,
			},
		},
		{
			msg: "invalid policy timeout",
			retryConfig: whitespace.Expand(`
				policies:
					realpolicy:
						retries: 1
						maxtimeout: abc
						backoff:
							exponential:
								first: 10ms
								max: 1s
				default: realpolicy
			`),
			wantError: []string{
				`error decoding`,
				`realpolicy`,
			},
		},
		{
			msg: "invalid policy backoff",
			retryConfig: whitespace.Expand(`
				policies:
					realpolicy:
						retries: 1
						maxtimeout: abc
						backoff: error
				default: realpolicy
			`),
			wantError: []string{
				`error decoding`,
				`realpolicy`,
			},
		},
		{
			msg:                "fallsback to default policy",
			retryConfig:        whitespace.Expand(``),
			wantPolicyProvider: newPolicyProviderBuilder().registerDefault(&defaultPolicy).provider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			var data map[string]interface{}
			err := yaml.Unmarshal([]byte(tt.retryConfig), &data)
			require.NoError(t, err, "error unmarshalling")

			policyProvider, err := NewPolicyProvider(data)
			if len(tt.wantError) > 0 {
				require.Error(t, err, "expected error, got none")
				for _, wantErr := range tt.wantError {
					require.Contains(t, err.Error(), wantErr, "expected error")
				}
				return
			}
			require.NoError(t, err, "error decoding")

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
	assert.Equal(t, expected.retries, actual.retries, "did not match on retries")
	assert.Equal(t, expected.maxRequestTimeout, actual.maxRequestTimeout, "did not match on maxRequestTimeout")

	expectedStrat, ok := expected.backoffStrategy.(*backoff.ExponentialStrategy)
	require.True(t, ok, "first strategy was not an exponential strategy")

	actualStrat, ok := actual.backoffStrategy.(*backoff.ExponentialStrategy)
	require.True(t, ok, "second strategy was not an exponential strategy")

	isEqual, msg := expectedStrat.IsEqual(actualStrat)
	assert.True(t, isEqual, msg)
}
