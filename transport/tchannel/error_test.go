// Copyright (c) 2025 Uber Technologies, Inc.
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

package tchannel

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestToYARPCError(t *testing.T) {
	tests := []struct {
		name    string
		giveErr error
		giveReq *transport.Request
		wantErr error
	}{
		{
			name:    "nil",
			giveErr: nil,
			wantErr: nil,
		},
		{
			name:    "yarpcerror",
			giveErr: yarpcerrors.InvalidArgumentErrorf("test"),
			wantErr: yarpcerrors.InvalidArgumentErrorf("test"),
		},
		{
			name:    "tchannel error",
			giveErr: tchannel.NewSystemError(tchannel.ErrCodeBadRequest, "test"),
			wantErr: fromSystemError(tchannel.NewSystemError(tchannel.ErrCodeBadRequest, "test").(tchannel.SystemError)),
		},
		{
			name:    "deadline exceeded",
			giveErr: context.DeadlineExceeded,
			giveReq: &transport.Request{Service: "serv", Procedure: "proc"},
			wantErr: yarpcerrors.DeadlineExceededErrorf("deadline exceeded for service: %q, procedure: %q", "serv", "proc"),
		},
		{
			name:    "unknown",
			giveErr: errors.New("test"),
			giveReq: &transport.Request{Service: "serv", Procedure: "proc"},
			wantErr: yarpcerrors.UnknownErrorf("received unknown error calling service: %q, procedure: %q, err: %s", "serv", "proc", "test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := toYARPCError(tt.giveReq, tt.giveErr)
			assert.Equal(t, tt.wantErr, gotErr)
		})
	}
}

func TestGetResponseErrorMeta(t *testing.T) {
	tests := []struct {
		name string
		give error
		want *ResponseErrorMeta
	}{
		{
			name: "nil",
		},
		{
			name: "wrong error",
			give: errors.New("not a yarpc/tchannel error"),
		},
		{
			name: "success",
			give: fromSystemError(tchannel.NewSystemError(tchannel.ErrCodeProtocol, "foo bar").(tchannel.SystemError)),
			want: &ResponseErrorMeta{
				Code: tchannel.ErrCodeProtocol,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetResponseErrorMeta(tt.give), "unexpected")
		})
	}
}
