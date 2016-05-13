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
	BlahBlah(req *thrift.Request) (*thrift.Response, error)
	SecondtestString(req *thrift.Request, thing *string) (string, *thrift.Response, error)
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

func (h handler) BlahBlah(req *thrift.Request, body wire.Value) (wire.Value, *thrift.Response, error) {
	var args secondservice.BlahBlahArgs
	if err := args.FromWire(body); err != nil {
		return wire.Value{}, nil, err
	}
	res, err := h.impl.BlahBlah(req)
	result, err := secondservice.BlahBlahHelper.WrapResponse(err)
	return result.ToWire(), res, err
}

func (h handler) SecondtestString(req *thrift.Request, body wire.Value) (wire.Value, *thrift.Response, error) {
	var args secondservice.SecondtestStringArgs
	if err := args.FromWire(body); err != nil {
		return wire.Value{}, nil, err
	}
	success, res, err := h.impl.SecondtestString(req, args.Thing)
	result, err := secondservice.SecondtestStringHelper.WrapResponse(success, err)
	return result.ToWire(), res, err
}
