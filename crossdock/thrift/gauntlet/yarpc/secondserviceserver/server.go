// Code generated by thriftrw

// Copyright (c) 2016 Uber Technologies, Inc.
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

package secondserviceserver

import (
	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/service/secondservice"
	"github.com/yarpc/yarpc-go/encoding/thrift"
)

type Interface interface {
	BlahBlah(reqMeta *thrift.ReqMeta) (*thrift.ResMeta, error)
	SecondtestString(reqMeta *thrift.ReqMeta, thing *string) (string, *thrift.ResMeta, error)
}

func New(impl Interface) thrift.Service {
	return service{handler{impl}}
}

type service struct{ h handler }

func (service) Name() string {
	return "SecondService"
}

func (service) Protocol() protocol.Protocol {
	return protocol.Binary
}

func (s service) Handlers() map[string]thrift.Handler {
	return map[string]thrift.Handler{"blahBlah": thrift.HandlerFunc(s.h.BlahBlah), "secondtestString": thrift.HandlerFunc(s.h.SecondtestString)}
}

type handler struct{ impl Interface }

func (h handler) BlahBlah(reqMeta *thrift.ReqMeta, body wire.Value) (thrift.Response, error) {
	var args secondservice.BlahBlahArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}
	resMeta, err := h.impl.BlahBlah(reqMeta)
	hadError := err != nil
	result, err := secondservice.BlahBlahHelper.WrapResponse(err)
	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body, err = result.ToWire()
	}
	return response, err
}

func (h handler) SecondtestString(reqMeta *thrift.ReqMeta, body wire.Value) (thrift.Response, error) {
	var args secondservice.SecondtestStringArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}
	success, resMeta, err := h.impl.SecondtestString(reqMeta, args.Thing)
	hadError := err != nil
	result, err := secondservice.SecondtestStringHelper.WrapResponse(success, err)
	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body, err = result.ToWire()
	}
	return response, err
}
