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

// Package pally is a simple, atomic-based metrics library. It interoperates
// seamlessly with both Prometheus and Tally, providing ready-to-use Prometheus
// text and Protocol Buffer endpoints, differential updates to StatsD- or
// M3-based systems, and excellent performance along the hot path.
//
// Metric Names
//
// Pally requires that all metric names, label names, and label values be valid
// both in Tally and in Prometheus. Metric and label names must pass
// IsValidName. Statically-defined label values must pass IsValidLabelValue,
// but dynamic label values are automatically scrubbed using ScrubLabelValue.
// This minimizes magic while still permitting use of label values generated at
// runtime (e.g., service names).
//
// Counters And Gauges
//
// Pally offers two simple metric types: counters and gauges. Counters
// represent an ever-accumulating total, like a car's odometer. Gauges
// represent a point-in-time measurement, like a car's speedometer. In Pally,
// both counters and gauges must have all their labels specified ahead of time.
//
// Vectors
//
// In many real-world situations, it's impossible to know all the labels for a
// metric ahead of time. For example, you may want to track the number of
// requests your server receives by caller; in most cases, you can't list all
// the possible callers ahead of time. To accommodate these situations, Pally
// offers vectors.
//
// Vectors represent a collection of metrics that have some constant labels,
// but some labels assigned at runtime. At vector construction time, you must
// specify the variable label keys; in our example, we'd supply "caller_name"
// as the only variable label. At runtime, pass the values for the variable
// labels to the Get (or MustGet) method on the vector. The number of label
// values must match the configured number of label keys, and they must be
// supplied in the same order. Vectors create metrics on demand, caching them
// for efficient repeated access.
//
//   registry := NewRegistry()
//   requestsByCaller := registry.NewCounterVector(Opts{
//     Name: "requests",
//     Help: "Total requests by caller name.",
//     ConstLabels: Labels{
//       "zone": "us-west-1",
//       "service": "my_service_name",
//     },
//     // At runtime, we'll supply the caller name.
//     VariableLabels: []string{"caller_name"},
//   })
//   // In real-world use, we'd do this in a handler function (and we'd
//   // probably use the safer Get variant).
//   vec.MustGet("some_calling_service").Inc()
//
package pally
