// Copyright (c) 2019 Uber Technologies, Inc.
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

// Package yarpcconfig implements a generic configuration system that may be
// used to build YARPC Dispatchers from configurations specified in different
// markup formats.
//
// Usage
//
// To build a Dispatcher, first create a new Configurator. This object will be
// responsible for loading your configuration. It does not yet know about the
// different transports, peer lists, etc. that you want to use. You can inform
// the Configurator about the different transports, peer lists, etc. by
// registering them using RegisterTransport, RegisterPeerChooser,
// RegisterPeerList, and RegisterPeerListUpdater.
//
// 	cfg := config.New()
// 	cfg.MustRegisterTransport(http.TransportSpec())
// 	cfg.MustRegisterPeerList(roundrobin.Spec())
//
// This object is re-usable and may be stored as a singleton in your
// application. Custom transports, peer lists, and peer list updaters may be
// integrated with the configuration system by registering more
// TransportSpecs, PeerChooserSpecs, PeerListSpecs, and PeerListUpdaterSpecs
// with it.
//
// Use LoadConfigFromYAML to load a yarpc.Config from YAML and pass that to
// yarpc.NewDispatcher.
//
// 	c, err := cfg.LoadConfigFromYAML("myservice", yamlConfig)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	dispatcher := yarpc.NewDispatcher(c)
//
// If you have already parsed your configuration from a different format, pass
// the parsed data to LoadConfig instead.
//
// 	var m map[string]interface{} = ...
// 	c, err := cfg.LoadConfig("myservice", m)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	dispatcher := yarpc.NewDispatcher(c)
//
// NewDispatcher or NewDispatcherFromYAML may be used to get a
// yarpc.Dispatcher directly instead of a yarpc.Config.
//
// 	dispatcher, err := cfg.NewDispatcherFromYAML("myservice", yamlConfig)
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
// The configuration accepts the following top-level attributes: transports,
// inbounds, and outbounds.
//
// 	inbounds:
// 	  # ...
// 	outbounds:
// 	  # ...
// 	transports:
// 	  # ...
// 	logging:
// 	  # ...
//
// See the following sections for details on the logging, transports,
// inbounds, and outbounds keys in the configuration.
//
// Inbound Configuration
//
// The 'inbounds' attribute configures the different ways in which the service
// receives requests. It is represented as a mapping between inbound transport
// type and its configuration. For example, the following states that we want
// to receive requests over HTTP.
//
// 	inbounds:
// 	  http:
// 	    address: :8080
//
// (For details on the configuration parameters of individual transport types,
// check the documentation for the corresponding transport package.)
//
// If you want multiple inbounds of the same type, specify a different name
// for it and add a 'type' attribute to its configuration:
//
// 	inbounds:
// 	  http:
// 	    address: :8081
// 	  http-deprecated:
// 	    type: http
// 	    address: :8080
//
// Any inbound can be disabled by adding a 'disabled' attribute.
//
// 	inbounds:
// 	  http:
// 	    address: :8080
// 	  http-deprecated:
// 	    type: http
// 	    disabled: true
// 	    address: :8081
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
// make Unary requests to keyvalue service over TChannel and Oneway requests over
// HTTP.
//
// 	keyvalue:
// 	  unary:
// 	    tchannel:
//        peer: 127.0.0.1:4040
// 	  oneway:
// 	    http:
//        url: http://127.0.0.1:8080/
//
// For convenience, if there is only one outbound configuration for a service,
// it may be specified one level higher (without the 'unary' or 'oneway'
// attributes). In this case, that transport will be used to send requests for
// all compatible RPC types. For example, the HTTP transport supports both,
// Unary and Oneway RPC types so the following states that requests for both
// RPC types must be made over HTTP.
//
// 	keyvalue:
// 	  http:
// 	    url: http://127.0.0.1:8080/
//
// Similarly, the following states that we only make Oneway requests to the
// "email" service and those are always made over HTTP.
//
// 	email:
// 	  http:
// 	    url: http://127.0.0.1:8080/
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
// Peer Configuration
//
// Transports that support peer management and selection through YARPC accept
// some additional keys in their outbound configuration.
//
// An explicit peer may be specified for a supported transport by using the
// `peer` option.
//
// 	keyvalue:
// 	  tchannel:
// 	    peer: 127.0.0.1:4040
//
// All requests made to this outbound will be made through this peer.
//
// If a peer list was registered with the system, the name of the peer list
// may be used to specify a more complex peer selection and load balancing
// strategy.
//
// 	keyvalue:
// 	  http:
// 	    url: https://host/yarpc
// 	    round-robin:
// 	      peers:
// 	        - 127.0.0.1:8080
// 	        - 127.0.0.1:8081
// 	        - 127.0.0.1:8082
//
// In the example above, the system will round-robin the requests between the
// different addresses. In case of the HTTP transport, the URL will be used as
// a template for the HTTP requests made to these hosts.
//
// Finally, the TransportSpec for a Transport may include named presets for
// peer lists in its definition. These may be referenced by name in the config
// using the `with` key.
//
// 	# Given a preset "dev-proxy" that was included in the TransportSpec, the
// 	# following is valid.
// 	keyvalue:
// 	  http:
// 	    url: https://host/yarpc
// 	    with: dev-proxy
//
// Transport Configuration
//
// The 'transports' attribute configures the Transport objects that are shared
// between all inbounds and outbounds of that transport type. It is
// represented as a mapping between the transport name and its configuration.
//
// 	transports:
// 	  http:
// 	    keepAlive: 30s
//
// (For details on the configuration parameters of individual transport types,
// check the documentation for the corresponding transport package.)
//
// Logging Configuration
//
// The 'logging' attribute configures how YARPC's observability middleware
// logs output.
//
// 	logging:
// 	  levels:
// 	    # ...
//
// The following keys are supported under the 'levels' key,
//
//  success
//    Configures the level at which successful requests are logged.
//    Defaults to "debug".
// 	applicationError
// 	  Configures the level at which application errors are logged.
//    All Thrift exceptions are considered application errors.
//    Defaults to "error".
//  failure
//    Configures the level at which all other failures are logged.
//    Default is "error".
//
// For example, the following configuration will have the effect of logging
// Thrift exceptions for inbound and outbound calls ("Error handling inbound
// request" and "Error making outbound call") at info level instead of error.
//
// 	logging:
// 	  levels:
// 	    applicationError: info
//
// The 'logging' attribute also has 'inbound' and 'outbound' sections
// to specify log levels that depend on the traffic direction.
// For example, the following configuration will only override the log level
// for successful outbound requests.
//
//  logging:
//    levels:
//      inbound:
//        success: debug
//
// The log levels are:
//
//  debug
//  info
//  warn
//  error
//  dpanic
//  panic
//  fatal
//
// Customizing Configuration
//
// When building your own TransportSpec, PeerListSpec, or PeerListUpdaterSpec,
// you will define functions accepting structs or pointers to structs which
// define the different configuration parameters needed to build that entity.
// These configuration parameters will be decoded from the user-specified
// configuration using a case-insensitive match on the field names.
//
// Given the struct,
//
// 	type MyConfig struct {
// 		URL string
// 	}
//
// An object containing a `url`, `URL` or `Url` key with a string value will
// be accepted in place of MyConfig.
//
// Configuration structs can use standard Go primitive types, time.Duration,
// maps, slices, and other similar structs. For example only, an outbound
// might accept a config containing an array of host:port structs (in
// practice, an outbound would use a config.PeerList to build a peer.Chooser).
//
// 	type Peer struct {
// 		Host string
// 		Port int
// 	}
//
// 	type MyOutboundConfig struct{ Peers []Peer }
//
// The above will accept the following YAML:
//
// 	myoutbound:
// 	  peers:
// 	    - host: localhost
// 	      port: 8080
// 	    - host: anotherhost
// 	      port: 8080
//
// Field names can be changed by adding a `config` tag to fields in the
// configuration struct.
//
// 	type MyInboundConfig struct {
// 		Address string `config:"addr"`
// 	}
//
// This struct will accept the `addr` key, not `address`.
//
// In addition to specifying the field name, the `config` tag may also include
// an `interpolate` option to request interpolation of variables in the form
// ${NAME} or ${NAME:default} at the time the value is decoded. By default,
// environment variables are used to fill these variables; this may be changed
// with the InterpolationResolver option.
//
// Interpolation may be requested only for primitive fields and time.Duration.
//
// 	type MyConfig struct {
// 		Address string `config:"addr,interpolate"`
// 		Timeout time.Duration `config:",interpolate"`
// 	}
//
// Note that for the second field, we don't change the name with the tag; we
// only indicate that we want interpolation for that variable.
//
// In the example above, values for both, Address and Timeout may contain
// strings in the form ${NAME} or ${NAME:default} anywhere in the value. These
// will be replaced with the value of the environment variable or the default
// (if specified) if the environment variable was unset.
//
// 	addr: localhost:${PORT}
// 	timeout: ${TIMEOUT_SECONDS:5}s
package yarpcconfig
