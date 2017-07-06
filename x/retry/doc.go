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
// a `map[string]interface{}` and pass it into the
// `NewOutboundMiddlewareFromConfig` function.
//
//  var data map[string]interface{}
//  err := yaml.Unmarshal(myYAMLConfig, &data)
//  mw, err := retry.NewOutboundMiddlewareFromConfig(data)
//
// Retry middleware can also be built by creating a PolicyProvider and passing
// it in as an option to the `NewOutboundMiddleware` function.
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
// The 'policies' attribute is a map from a policy name to the configuration for
// that policy.
//
//  policies:
//    fastretry:
//      retries: 5
//      maxTimeout: 10ms
//      backoff:
//        exponential:
//          first: 5ms
//          max: 1s
//    slowretry:
//      retries: 3
//      maxTimeout: 100ms
//      backoff:
//        exponential:
//          first: 5ms
//          max: 10s
//
// This configuration would create two retry policies we could use later.
//
// The fastretry policy will enforce a per-request timeout of 10 milliseconds,
// making 5 more attempts after the first failure, with an exponential backoff
// starting at 5 milliseconds between requests and a maximum of 1 second.
//
// The slowretry policy will enforce a per-request timeout of 100 milliseconds,
// and will make 3 more attempts after the first failure, with an exponential
// backoff starting at 5 milliseconds between requests and a maximum of 10
// seconds.
//
// Default Configuration
//
// The 'default' attributes indicates which policy will be the default for
// all retry attempts.  The value must be a policy name from the 'policies'
// section, if unspecified, no retries will be performed by default.
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
// attribute, which must point to one of the 'policies' we've defined in the
// 'policies' section.
// We can specify policies with varying levels of granularity. We can specify
// both, 'service' and 'procedure' to apply the policy to requests made to that
// procedure of that service, or we can specify just 'service' to apply the
// given policy to all requests made to that service.
//
// In terms of preference, the order of importance for policies will be:
//
//   1) "service" and "procedure" overrides
//   2) "service" overrides
//   3) default policy
package retry
