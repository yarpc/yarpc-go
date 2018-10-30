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
	"go.uber.org/yarpc/v2/yarpchttp"
	"go.uber.org/yarpc/v2/yarpcrandpeer"
	"go.uber.org/yarpc/v2/yarpcrouter"
)

type lifecycle interface {
	Start(context.Context) error
	Stop(context.Context) error
}

func newProcedures() []yarpc.TransportProcedure {
	return jsonProcedures()
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
		pl := yarpcrandpeer.New(dialer)
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

func TestGuantlet(t *testing.T) {
	transports := []string{_http}
	encodings := []string{_json}
	choosers := []string{_random}

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

				// call with encoding clients
				for _, encoding := range encodings {
					t.Run(encoding, func(t *testing.T) {
						var resHeaders map[string]string
						callOptions := newCallOptions(&resHeaders)

						switch encoding {
						case _json:
							validateJSON(t, client, callOptions)
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
