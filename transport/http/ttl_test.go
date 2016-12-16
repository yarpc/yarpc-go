package http

import (
	"context"
	"testing"

	"go.uber.org/yarpc/api/transport"

	"github.com/stretchr/testify/assert"
)

func TestParseTTL(t *testing.T) {
	req := &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Procedure: "hello",
		Encoding:  "raw",
	}

	tests := []struct {
		ttlString   string
		wantErr     error
		wantMessage string
	}{
		{ttlString: "1"},
		{
			ttlString: "-1000",
			wantErr: invalidTTLError{
				Service:   "service",
				Procedure: "hello",
				TTL:       "-1000",
			},
			wantMessage: `invalid TTL "-1000" for procedure "hello" of service "service": must be positive integer`,
		},
		{
			ttlString: "not an integer",
			wantErr: invalidTTLError{
				Service:   "service",
				Procedure: "hello",
				TTL:       "not an integer",
			},
			wantMessage: `invalid TTL "not an integer" for procedure "hello" of service "service": must be positive integer`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.ttlString, func(t *testing.T) {
			ctx, cancel, err := parseTTL(context.Background(), req, tt.ttlString)
			defer cancel()

			if tt.wantErr != nil && assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err)
				assert.Equal(t, tt.wantMessage, err.Error())
			} else {
				assert.NoError(t, err)
				_, ok := ctx.Deadline()
				assert.True(t, ok)
			}
		})
	}
}
