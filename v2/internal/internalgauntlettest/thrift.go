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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internalgauntlettest/thrift/echo"
	"go.uber.org/yarpc/v2/internal/internalgauntlettest/thrift/echo/echoclient"
	"go.uber.org/yarpc/v2/internal/internalgauntlettest/thrift/echo/echoserver"
	"go.uber.org/yarpc/v2/yarpcthrift"
)

type thriftHandler struct{}

func thriftProcedures() []yarpc.TransportProcedure {
	return echoserver.New(thriftHandler{})
}

func (thriftHandler) Echo(ctx context.Context, request *echo.EchoRequest) (*echo.EchoResponse, error) {
	call := yarpc.CallFromContext(ctx)
	err := validateCallOptions(call, yarpcthrift.Encoding)
	if err != nil {
		return nil, err
	}

	err = call.WriteResponseHeader(_headerKeyRes, _headerValueRes)
	if err != nil {
		return nil, err
	}

	return &echo.EchoResponse{Message: request.Message}, nil
}

func validateThrift(t *testing.T, client yarpc.Client, callOptions []yarpc.CallOption) {
	msg := "hello world!! (╯°□°)╯︵ ┻━┻"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := echoclient.New(client).Echo(
		ctx,
		&echo.EchoRequest{Message: msg},
		callOptions...)

	require.NoError(t, err, "error making call")
	assert.Equal(t, msg, res.Message)
}
