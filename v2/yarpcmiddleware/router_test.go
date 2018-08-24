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

package yarpcmiddleware

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcmiddlewaretest"
	"go.uber.org/yarpc/v2/yarpctransporttest"
)

func TestApplyRouteTable(t *testing.T) {
	ctrl := gomock.NewController(t)

	routerMiddleware := yarpcmiddlewaretest.NewMockRouter(ctrl)
	routeTable := yarpctransporttest.NewMockRouteTable(ctrl)

	rtWithMW := ApplyRouteTable(routeTable, routerMiddleware)

	routeTableWithMW, ok := rtWithMW.(routeTableWithMiddleware)
	require.True(t, ok, "unexpected RouteTable type")

	assert.Equal(t, routeTableWithMW.m, routerMiddleware)
	assert.Equal(t, routeTableWithMW.r, routeTable)
}

func TestRouteTableMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	routerMiddleware := yarpcmiddlewaretest.NewMockRouter(ctrl)
	routeTable := yarpctransporttest.NewMockRouteTable(ctrl)

	ctx := context.Background()
	req := &yarpc.Request{}
	spec := yarpc.NewUnaryHandlerSpec(yarpctransporttest.NewMockUnaryHandler(ctrl))

	routeTableWithMW := ApplyRouteTable(routeTable, routerMiddleware)

	// register procedures
	routeTable.EXPECT().Register(gomock.Any())
	routeTableWithMW.Register([]yarpc.Procedure{})

	// choose handlerSpec
	routerMiddleware.EXPECT().Choose(ctx, req, routeTable).Return(spec, nil)
	spec, err := routeTableWithMW.Choose(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, spec, spec)

	// get procedures
	routerMiddleware.EXPECT().Procedures(routeTable).Return([]yarpc.Procedure{})
	routeTableWithMW.Procedures()
}
