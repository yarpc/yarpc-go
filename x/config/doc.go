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
// different transports that it needs to support. This object is re-usable
// and may be stored as a singleton in your application.
//
// 	cfg := config.New()
// 	cfg.MustRegisterTransport(http.TransportSpec())
// 	cfg.MustRegisterTransport(redis.TransportSpec())
//
// Use LoadConfigFromYAML to load a yarpc.Config from YAML and pass that to
// yarpc.NewDispatcher.
//
// 	c, err := cfg.LoadConfigFromYAML(yamlConfig)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	dispatcher := yarpc.NewDispatcher(c)
//
// If you have already parsed your configuration from a different format, pass
// the parsed data to LoadConfig instead.
//
// 	var m map[string]interface{} = ...
// 	c, err := cfg.LoadConfig(m)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	dispatcher := yarpc.NewDispatcher(c)
//
// NewDispatcher or NewDispatcherFromYAML may be used to get a
// yarpc.Dispatcher directly instead of a yarpc.Config.
//
// 	dispatcher, err := cfg.NewDispatcherFromYAML(yamlConfig)
//
// Configuration parameters for the different transports, inbounds, and
// outbounds are defined in the TransportSpecs that were registered against
// the Configurator. A TransportSpec uses this information to build the
// corresponding Transport, Inbound and Outbound objects.
//
// Configuration
//
// The configuration may be specified in YAML or as any Go-level
// map[string]interface{}. The examples below use YAML for illustration
// purposes but other markup formats may be parsed into map[string]interface{}
// as long as the information provided is the same.
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
// attributes). In this case, all RPC types supported by that transport will
// be set. For example, the HTTP transport supports both, Unary and Oneway RPC
// types so the following states that requests for both RPC types must be made
// over HTTP.
//
// 	keyvalue:
// 	  http:
// 	    # ...
//
// Similarly, the following states that we only make Oneway requests to the
// "email" service and those are always made over Redis.
//
// 	email:
// 	  redis:
// 	    # ...
//
// When the name of the target service differs from the outbound name, it may
// be overridden with the 'service' key.
//
// 	keyvalue:
// 	  unary:
// 	    # ...
// 	  oneway:
// 	    # ...
// 	keyvalue-staging:
// 	  service: keyvalue
// 	  unary:
// 	    # ...
// 	  oneway:
// 	    # ...
//
// Transport Configuration
//
// The 'transports' attribute configures the Transport objects that are shared
// between all inbounds and outbounds of that transport type. It is
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
// 		BuildTransport: func(*myTransportConfig) (transport.Transport, error) {
// 			// ...
// 		},
// 		...
// 	})
//
// This transport will be configured under the 'mytransport' key in the
// parsed configuration data. See documentation for TransportSpec for details
// on what each field of TransportSpec means and how it behaves.
package config
