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

// Package retry provides a YARPC middleware which is able to retry failed
// outbound unary requests.
//
// Usage
//
// To build a retry middleware from config, first decode your configuration into
// a `map[string]interface{}`
//
//  var data map[string]interface{}
//  err := yaml.Unmarshal(myYAMLConfig, &data)
//  mw, err := retry.NewOutboundMiddlewareFromConfig(data)
//
// Retry middleware can also be built by creating a PolicyProvider and passing
// it in as an option.
//
//  mw := retry.NewOutboundMiddleware(retry.WithPolicyProvider(policyProvider))
//
// Check out the PolicyProvider docs for more details.
//
// Configuration
//
// The configuration may be specified in YAML or as any Go-level
// map[string]interface{}. The examples below use YAML for illustration
// purposes but other markup formats may be parsed into map[string]interface{}
// as long as the information provided is the same.
//
// The configuration accepts the following top-level attributes: policies,
// default, and overrides.
//
//  policies:
//    # ...
//  default: ...
//  overrides:
//    # ...
//
// See the following sections for details on the policies, default, and
// overrides keys in the configuration.
//
// Policies Configuration
//
// The 'policies' attribute configures different retry policies which can be
// referenced individually in the 'default' and 'override' sections.
//
//  policies:
//    fastretry:
//      retries: 5
//      maxtimeout: 10ms
//      backoff:
//        exponential:
//          first: 5ms
//          max: 1s
//    slowretry:
//      retries: 3
//      maxtimeout: 100ms
//      backoff:
//        exponential:
//          first: 5ms
//          max: 10s
//
// This configuration would create two retry policies we could use later.
//
// "fastretry" will max out request timeouts to 10 ms, and will re-attempt
// requests 5 times before failing.  Between every failure, it will initially
// use a full jitter exponential backoff of 5ms with a max backoff of 1s.
//
// "slowretry" will max out request timeouts to 100 ms, and will re-attempt
// requests 3 times before failing.  Between every failure, it will initially
// use a full jitter exponential backoff of 5ms with a max backoff of 10s.
//
// Default Configuration
//
// The 'default' attributes indicates which "policy" will be the default for
// all retry attempts.
//
// 	default: slowretry
//
// 'slowretry' needs to be a defined policy in the 'policies' section (or it
// needs to be one of the default policies we've provided).
//
// Overrides Configuration
//
// The 'overrides' attribute configures custom retry policies for specific
// services or procedures.
//
//  overrides:
//    - service: fastservice
//      with: fastretry
//    - service: fastservice
//      procedure: slowprocedure
//      with: slowretry
//
// Each override specifies which policy will be used based on the `with`
// attribute, which must point to one of the 'policies' we've predefined.
// Additionally, we can specify which "service" the policy will be applied to.
// or which "service+procedure" combination the policy will be applied to.
//
// In terms of preference, the order of importance for policies will be:
//
//   1) "service"+"procedure" overrides
//   2) "service" overrides
//   3) default policy
package retry
