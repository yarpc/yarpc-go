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

package cherami

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/config"

	"github.com/uber/cherami-client-go/client/cherami"
)

const (
	_destinationSuffix   = "yarpc_dest"
	_consumerGroupSuffix = "yarpc_cg"
)

var errNoPeerList = errors.New(
	"cannot automatically discover Cherami: " +
		"no peer list provided: " +
		"please set the peerList attribute or " +
		"provide a default peer list using the DefaultPeerList option")

// TransportSpecOption configures the Cherami TransportSpec.
type TransportSpecOption func(*transportSpec)

// DefaultPeerList specifies the default path at which the TChannel peer list
// may be found.
//
// This value will be used when building Cherami transports that automatically
// discover Cherami Frontends if the user did not provide their own peer list.
func DefaultPeerList(path string) TransportSpecOption {
	return func(ts *transportSpec) {
		ts.defaultPeerList = path
	}
}

// TransportSpec builds a TransportSpec for the Cherami transport.
// TransportSpecOptions may be passed to this function to configure the
// behavior of the TransportSpec.
//
// 	configurator.MustRegisterTransport(
// 		cherami.TransportSpec(cherami.DefaultPeerList("/etc/hosts.json")),
// 	)
//
// See TransportConfig, InboundConfig, and OutboundConfig for details on the
// different configuration parameters supported by this Transport.
func TransportSpec(opts ...TransportSpecOption) config.TransportSpec {
	var ts transportSpec
	for _, opt := range opts {
		opt(&ts)
	}
	return ts.Spec()
}

// When building inbounds and outbounds, instead of casting
// transport.Transport to *Transport, we'll cast to the interface
// cheramiTransport so that we can test against a mock cheramiTransport.
type cheramiTransport interface {
	transport.Transport

	NewInbound(InboundOptions) *Inbound
	NewOutbound(OutboundOptions) *Outbound
}

var _ cheramiTransport = (*Transport)(nil)

// TransportSpec holds the configurable parts of the Cherami TransportSpec.
type transportSpec struct {
	defaultPeerList string
}

func (ts *transportSpec) Spec() config.TransportSpec {
	return config.TransportSpec{
		Name:                "cherami",
		BuildTransport:      ts.buildTransport,
		BuildInbound:        ts.buildInbound,
		BuildOnewayOutbound: ts.buildOnewayOutbound,
	}
}

// TransportConfig configures the shared Cherami Transport. This is shared
// between all Cherami outbounds and inbounds of a Dispatcher.
//
// All fields in TransportConfig are optional and this section may be skipped
// entirely for most use cases.
type TransportConfig struct {
	// If specified, this is the address of a specific Cherami Frontend or a
	// locally hosted development instance. Most users should omit this since
	// the Cherami Frontend will be discovered automatically.
	//
	// 	address: 127.0.0.1:4922
	Address string `config:"address,interpolate"`

	// Path to a JSON file containing the TChannel peer list used to
	// auto-discover Cherami Frontend machines. This may be skipped if a
	// default peer list was provided on TransportSpec instantiation or if an
	// Address was provided.
	//
	// 	peerList: /etc/hosts.json
	PeerList string `config:"peerList,interpolate"`

	// Timeout for requests to the Cherami service. The default timeout should
	// suffice for most use cases.
	//
	// 	timeout: 5s
	Timeout time.Duration `config:"timeout"`

	// Name of the Cherami deployment to which requests should be sent. Some
	// valid values are, "prod", "staging", "staging2", and "dev".
	//
	// 	deploymentStr: dev
	//
	// Defaults to "prod".
	DeploymentStr string `config:"deploymentStr,interpolate"`
}

// Parses the IP address and port from the given address.
func parseIPAndPort(address string) (ip string, port int, _ error) {
	if address == "" {
		return ip, port, errors.New("address is unspecified")
	}

	idx := strings.LastIndexByte(address, ':')
	if idx == -1 {
		return ip, port, fmt.Errorf("invalid address %q: port was not specified", address)
	}

	ip = address[:idx]
	portStr := address[idx+1:]
	port64, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return ip, port, fmt.Errorf("invalid port %q in address %q: %v", portStr, address, err)
	}

	return ip, int(port64), nil
}

func (ts *transportSpec) buildTransport(
	tc *TransportConfig, kit *config.Kit,
) (transport.Transport, error) {
	opts := cherami.ClientOptions{DeploymentStr: tc.DeploymentStr, Timeout: tc.Timeout}

	var client cherami.Client
	switch {
	case len(tc.Address) > 0: // Explicit Cherami frontend
		ip, port, err := parseIPAndPort(tc.Address)
		if err != nil {
			return nil, err
		}

		client, err = cherami.NewClient(kit.ServiceName(), ip, int(port), &opts)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create Cherami client with address %q: %v", tc.Address, err)
		}
	case len(tc.PeerList) > 0 || len(ts.defaultPeerList) > 0: // Auto-discover Cherami
		peerList := ts.defaultPeerList
		if len(tc.PeerList) > 0 {
			peerList = tc.PeerList
		}

		var err error
		client, err = cherami.NewHyperbahnClient(kit.ServiceName(), peerList, &opts)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create Cherami client with peer list %q: %v", peerList, err)
		}
	default:
		return nil, errors.New("either an `address` or a `peerList` must be specified")
	}

	return NewTransport(client), nil
}

// InboundConfig configures a Cherami Inbound.
//
// 	inbounds:
// 	  cherami:
// 	    destination: /myservice/yarpc_dest
// 	    consumerGroup: /myservice/yarpc_cg
type InboundConfig struct {
	// Destination from which RPCs for this inbound will be retrieved.
	//
	// If unspecified, the destination "/${service}/yarpc_dest" will be used
	// where ${service} is the name of your YARPC service.
	Destination string `config:"destination,interpolate"`

	// Name of the consumer group used to read RPCs from Cherami.
	//
	// If unspecified, the consumer group "/${service}/yarpc_cg" will be used
	// where ${service} is the name of your YARPC service.
	ConsumerGroup string `config:"consumerGroup,interpolate"`

	// Number of requests to buffer locally. If requests are short-lived,
	// setting this to a higher value may improve throughput at the cost of
	// memory usage.
	//
	// Defaults to 10.
	PrefetchCount int `config:"prefetchCount"`
}

func (ts *transportSpec) buildInbound(
	tc *InboundConfig,
	t transport.Transport,
	kit *config.Kit,
) (transport.Inbound, error) {
	opts := InboundOptions{
		Destination:   tc.Destination,
		ConsumerGroup: tc.ConsumerGroup,
		PrefetchCount: tc.PrefetchCount,
	}

	if opts.Destination == "" {
		opts.Destination = fmt.Sprintf("/%v/%v", kit.ServiceName(), _destinationSuffix)
	}

	if opts.ConsumerGroup == "" {
		opts.ConsumerGroup = fmt.Sprintf("/%v/%v", kit.ServiceName(), _consumerGroupSuffix)
	}

	return t.(cheramiTransport).NewInbound(opts), nil
}

// OutboundConfig configures a Cherami Outbound.
type OutboundConfig struct {
	// Destination to which RPCs for this outbound will be written.
	//
	// If unspecified, the destination "/${service}/yarpc_dest" will be used
	// where ${service} is the name of the destination service.
	Destination string `config:"destination,interpolate"`
}

func (ts *transportSpec) buildOnewayOutbound(
	tc *OutboundConfig,
	t transport.Transport,
	kit *config.Kit,
) (transport.OnewayOutbound, error) {
	opts := OutboundOptions{Destination: tc.Destination}

	if opts.Destination == "" {
		opts.Destination = fmt.Sprintf("/%v/%v", kit.ServiceName(), _destinationSuffix)
	}

	return t.(cheramiTransport).NewOutbound(opts), nil
}
