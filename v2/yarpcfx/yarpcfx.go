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

package yarpcfx

import (
	"go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcclient"
	"go.uber.org/yarpc/v2/yarpcrouter"
)

const _name = "yarpcfx"

// Module provides YARPC integration for services. The module produces
// a yarpc.Router and a yarpc.ClientProvider.
var Module = fx.Options(
	fx.Provide(NewClientProvider),
	fx.Provide(NewRouter),
)

// ClientProviderParams defines the dependencies of this module.
type ClientProviderParams struct {
	fx.In

	Clients     []yarpc.Client   `group:"yarpcfx"`
	ClientLists [][]yarpc.Client `group:"yarpcfx"`
}

// ClientProviderResult defines the values produced by this module.
type ClientProviderResult struct {
	fx.Out

	Provider yarpc.ClientProvider
}

// NewClientProvider provides a yarpc.ClientProvider to the Fx application.
func NewClientProvider(p ClientProviderParams) (ClientProviderResult, error) {
	clients := p.Clients
	for _, cl := range p.ClientLists {
		clients = append(clients, cl...)
	}
	provider := yarpcclient.NewProvider()
	for _, c := range clients {
		provider.Register(c.Service, c)
	}
	return ClientProviderResult{
		Provider: provider,
	}, nil
}

// RouterParams defines the parameters for procedure registration and
// router construction.
type RouterParams struct {
	fx.In

	RouterMiddleware yarpc.RouterMiddleware       `optional:"true"`
	Procedures       []yarpc.TransportProcedure   `group:"yarpcfx"`
	ProcedureLists   [][]yarpc.TransportProcedure `group:"yarpcfx"`
}

// RouterResult defines the values produced by this module.
type RouterResult struct {
	fx.Out

	Router yarpc.Router
}

// NewRouter registers procedures with a router, and produces it so
// that specific transport inbounds can depend upon it.
func NewRouter(p RouterParams) (RouterResult, error) {
	procedures := p.Procedures
	for _, pl := range p.ProcedureLists {
		procedures = append(procedures, pl...)
	}
	router := yarpcrouter.NewMapRouter("foo" /* Derive from servicefx. */, procedures)
	return RouterResult{
		Router: yarpc.ApplyRouter(router, p.RouterMiddleware),
	}, nil
}
