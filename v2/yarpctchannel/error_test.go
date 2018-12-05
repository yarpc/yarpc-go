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

package yarpctchannel

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber/tchannel-go"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
)

func TestToYARPCError(t *testing.T) {
	tests := []struct {
		name    string
		giveErr error
		giveReq *yarpc.Request
		wantErr error
	}{
		{
			name:    "nil",
			giveErr: nil,
			wantErr: nil,
		},
		{
			name:    "yarpcerror",
			giveErr: yarpcerror.InvalidArgumentErrorf("test"),
			wantErr: yarpcerror.InvalidArgumentErrorf("test"),
		},
		{
			name:    "tchannel error",
			giveErr: tchannel.NewSystemError(tchannel.ErrCodeBadRequest, "test"),
			wantErr: fromSystemError(tchannel.NewSystemError(tchannel.ErrCodeBadRequest, "test").(tchannel.SystemError)),
		},
		{
			name:    "deadline exceeded",
			giveErr: context.DeadlineExceeded,
			giveReq: &yarpc.Request{Service: "serv", Procedure: "proc"},
			wantErr: yarpcerror.DeadlineExceededErrorf("deadline exceeded for service: %q, procedure: %q", "serv", "proc"),
		},
		{
			name:    "unknown",
			giveErr: errors.New("test"),
			giveReq: &yarpc.Request{Service: "serv", Procedure: "proc"},
			wantErr: yarpcerror.UnknownErrorf("error for service %q and procedure %q: %v", "serv", "proc", "test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := toYARPCError(tt.giveReq, tt.giveErr)
			assert.Equal(t, tt.wantErr, gotErr)
		})
	}
}
