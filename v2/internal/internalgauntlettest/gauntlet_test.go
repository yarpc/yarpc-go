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

package internalgauntlettest

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcgrpc"
	"go.uber.org/yarpc/v2/yarpchttp"
	"go.uber.org/yarpc/v2/yarpcpendingheap"
	"go.uber.org/yarpc/v2/yarpcrandpeer"
	"go.uber.org/yarpc/v2/yarpcroundrobin"
	"go.uber.org/yarpc/v2/yarpcrouter"
	"go.uber.org/yarpc/v2/yarpctchannel"
)

type lifecycle interface {
	Start(context.Context) error
	Stop(context.Context) error
}

func newProcedures() []yarpc.TransportProcedure {
	return append(append(jsonProcedures(), thriftProcedures()...), protoProcedures()...)
}

func newInbound(t *testing.T, transport string, listener net.Listener, procedures []yarpc.TransportProcedure) (stop func()) {
	var inbound lifecycle
	router := yarpcrouter.NewMapRouter(_service, procedures)

	switch transport {
	case _http:
		inbound = &yarpchttp.Inbound{
			Listener: listener,
			Router:   router,
		}

	case _gRPC:
		inbound = &yarpcgrpc.Inbound{
			Listener: listener,
			Router:   router,
		}

	case _tchannel:
		inbound = &yarpctchannel.Inbound{
			Service:  "service",
			Listener: listener,
			Router:   router,
		}

	default:
		t.Fatalf("unsupported transport: %q", transport)
	}

	require.NoError(t, inbound.Start(context.Background()))
	return func() { assert.NoError(t, inbound.Stop(context.Background()), "could not stop inbound") }
}

// returns a yarpc.Chooser with ID added to the backing peer list
func newChooser(t *testing.T, chooser string, dialer yarpc.Dialer, id yarpc.Identifier) yarpc.Chooser {
	update := yarpc.ListUpdates{Additions: []yarpc.Identifier{id}}

	switch chooser {
	case _random:
		pl := yarpcrandpeer.New("random", dialer)
		pl.Update(update)
		return pl

	case _roundrobin:
		pl := yarpcroundrobin.New("roundrobin", dialer)
		pl.Update(update)
		return pl

	case _pendingheap:
		pl := yarpcpendingheap.New("pending-heap", dialer)
		pl.Update(update)
		return pl

	default:
		t.Fatalf("unsupported peer chooser: %q", chooser)
	}

	return nil
}

// returns UnaryOutbounds configured with every specified chooser
// eg HTTP+round-robin, HTTP+random
func newOutbounds(t *testing.T, transport string, addr string, choosers []string) (_ []yarpc.UnaryOutbound, stop func()) {
	outbounds := make([]yarpc.UnaryOutbound, 0, len(choosers))
	dialers := make([]lifecycle, 0, len(choosers))

	id := yarpc.Address(addr)

	for _, chooser := range choosers {
		switch transport {
		case _http:
			dialer := &yarpchttp.Dialer{}
			require.NoError(t, dialer.Start(context.Background()))
			dialers = append(dialers, dialer)

			outbounds = append(outbounds, &yarpchttp.Outbound{
				Chooser: newChooser(t, chooser, dialer, id),
				URL:     &url.URL{Scheme: "http", Host: addr},
			})

		case _gRPC:
			dialer := &yarpcgrpc.Dialer{}
			require.NoError(t, dialer.Start(context.Background()))
			dialers = append(dialers, dialer)

			outbounds = append(outbounds, &yarpcgrpc.Outbound{
				Chooser: newChooser(t, chooser, dialer, id),
				URL:     &url.URL{Host: addr},
			})

		case _tchannel:
			dialer := &yarpctchannel.Dialer{
				Caller: "caller",
			}
			require.NoError(t, dialer.Start(context.Background()))
			dialers = append(dialers, dialer)

			outbounds = append(outbounds, &yarpctchannel.Outbound{
				Chooser: newChooser(t, chooser, dialer, id),
				Addr:    addr,
			})

		default:
			t.Fatalf("unsupported transport: %q", transport)
		}
	}

	stop = func() {
		for _, dialer := range dialers {
			assert.NoError(t, dialer.Stop(context.Background()))
		}
	}

	return outbounds, stop
}

func TestGauntlet(t *testing.T) {
	transports := []string{_http, _gRPC, _tchannel}
	encodings := []string{_json, _thrift, _proto}
	choosers := []string{_random, _roundrobin}

	procedures := newProcedures()

	for _, transport := range transports {
		t.Run(transport, func(t *testing.T) {
			// inbound
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			stop := newInbound(t, transport, listener, procedures)
			defer stop()

			// outbounds + choosers
			outbounds, stop := newOutbounds(t, transport, listener.Addr().String(), choosers)
			defer stop()

			for _, outbound := range outbounds {
				client := yarpc.Client{
					Caller:  _caller,
					Service: _service,
					Unary:   outbound,
				}

				for _, encoding := range encodings {
					t.Run(encoding, func(t *testing.T) {

						var resHeaders map[string]string
						callOptions := newCallOptions(&resHeaders)

						// call with encoding clients
						switch encoding {
						case _json:
							validateJSON(t, client, callOptions)
						case _thrift:
							validateThrift(t, client, callOptions)
						case _proto:
							validateProto(t, client, callOptions)
						default:
							t.Fatalf("unsupported encoding %s", encoding)
						}

						// validate response headers
						wantResponseHeaders := map[string]string{
							_headerKeyRes: _headerValueRes,
						}
						assert.Equal(t, wantResponseHeaders, resHeaders, "response headers did not match")
					})
				}
			}
		})
	}
}
