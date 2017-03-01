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

// Package config implements a generic configuration system that may be used
// to build YARPC Dispatchers from configurations specified in different
// markup formats.
//
// Usage
//
// To build a Dispatcher, set up a Configurator and inform it about the
// different transports that it needs to support. Use LoadConfig or
// LoadConfigFromYAML to load a yarpc.Config and pass that to
// yarpc.NewDispatcher.
//
// 	cfg := config.New()
// 	http.RegisterTransport(cfg)
// 	redis.RegisterTransport(cfg)
//
// 	c, err := cfg.LoadConfigFromYAML(yamlConfig)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	dispatcher := yarpc.NewDispatcher(c)
//
// Alternatively, use NewDispatcher or NewDispatcherFromYAML to build a
// Dispatcher directly.
//
// 	dispatcher, err := cfg.NewDispatcherFromYAML(yamlConfig)
//
// Configuration parameters for the different transports, inbounds, and
// outbounds are defined in the types fed into the TransportSpec that was
// registered against the Configurator.
//
// Configuration
//
// The configuration may be specified in YAML or as any Go-level
// map[string]interface{}. The examples below use YAML for illustration
// purposes.
//
// The configuration accepts the following top-level attributes: name,
// transports, inbounds, and outbounds.
//
// 	name: myservice
// 	inbounds:
// 	  # ...
// 	outbounds:
// 	  # ...
// 	transports:
// 	  # ...
//
// Where name specifies the name of the current service. See the following
// sections for details on the transports, inbounds, and outbounds keys in the
// configuration.
//
// Inbound Configuration
//
// The 'inbounds' attribute configures the different ways in which the service
// receives requests. It is represented as a mapping between inbound transport
// type and its configuration. For example, the following states that we want
// to receive requests over HTTP and Redis.
//
// 	inbounds:
// 	  redis:
// 	    # ...
// 	  http:
// 	    # ...
//
// (For details on the configuration parameters of individual transport types,
// check the documentation for the corresponding transport package.)
//
// If you want multiple inbounds of the same type, specify a different name
// for it and add a 'type' attribute to its configuration:
//
// 	inbounds:
// 	  http:
// 	    # ...
// 	  http-deprecated:
// 	    type: http
// 	    # ...
//
// Any inbound can be disabled by adding a 'disabled' attribute.
//
// 	inbounds:
// 	  http:
// 	    # ...
// 	  http-deprecated:
// 	    type: http
// 	    disabled: true
// 	    # ...
//
// Outbound Configuration
//
// The 'outbounds' attribute configures how this service makes requests to
// other YARPC-compatible services. It is represented as mapping between
// service name and outbound configuration.
//
// 	outbounds:
// 	  keyvalue:
// 	    # ..
// 	  anotherservice:
// 	    # ..
//
// (For details on the configuration parameters of individual transport types,
// check the documentation for the corresponding transport package.)
//
// The outbound configuration for a service has at least one of the following
// keys: unary, oneway. These specify the configurations for the corresponding
// RPC types for that service. For example, the following specifies that we
// make Unary requests to keyvalue service over HTTP and Oneway requests over
// Redis.
//
// 	keyvalue:
// 	  unary:
// 	    http:
// 	      # ...
// 	  oneway:
// 	    redis:
// 	      # ...
//
// For convenience, if there is only one outbound configuration for a service,
// it may be specified one level higher (without the 'unary' or 'oneway'
// attributes).
//
// 	keyvalue:
// 	  http:
// 	    # ...
//
// When the name of the target service differs from the outbound name, it may
// be overridden with the 'service' key.
//
// 	keyvalue-staging:
// 	  service: keyvalue
// 	  unary:
// 	    # ...
// 	  oneway:
// 	    # ...
//
// 	anotherservice-staging:
// 	  service: anotherservice
// 	  http:
// 	    # ...
//
// Transport Configuration
//
// The 'transports' attribute configures the global Transport objects that are
// shared between all inbounds and outbounds of that transport type. It is
// represented as a mapping between the transport name and its configuration.
//
// 	transports:
// 	  redis:
// 	    # ...
// 	  http:
// 	    # ...
//
// (For details on the configuration parameters of individual transport types,
// check the documentation for the corresponding transport package.)
//
// Defining a Transport
//
// To teach a Configurator about a Transport, register a TransportSpec against
// it.
//
// 	cfg.RegisterTransport(TransportSpec{
// 		Name: "mytransport",
// 		// ...
// 	})
//
// Configuration for this transport will be expected under the 'mytransport'
// type in the configuration. See TransportSpec for details on each field.
package config
