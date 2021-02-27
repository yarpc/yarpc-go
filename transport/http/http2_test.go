package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

func TestFromHTTP2ConnectRequest(t *testing.T) {
	tests := []struct {
		desc      string
		treq      *transport.Request
		wantError error
	}{
		{
			desc: "malformed CONNECT request: :scheme header set",
			treq: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{":scheme": "http2"}),
			},
			wantError: errMalformedHTTP2ConnectRequestExtraScheme,
		},
		{
			desc: "malformed CONNECT request: :path header set",
			treq: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{":path": "foo/path"}),
			},
			wantError: errMalformedHTTP2ConnectRequestExtraPath,
		},
		{
			desc:      "malformed CONNECT request: :authority header missing",
			treq:      &transport.Request{},
			wantError: errMalformedHTTP2ConnectRequestExtraAuthority,
		},
		{
			desc: "malformed CONNECT request: :authority header missing",
			treq: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{":authority": "127.0.0.1:1234"}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			req, err := fromHTTP2ConnectRequest(tt.treq)
			if tt.wantError != nil {
				assert.EqualError(t, err, tt.wantError.Error())
				return
			}
			assert.Equal(t, http.MethodConnect, req.Method)
		})
	}
}
