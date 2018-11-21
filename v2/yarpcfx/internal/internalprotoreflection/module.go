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

package internalprotoreflection

import (
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcprotobuf/reflection"
)

// Module provides reflection prodceudres for protobuf services.
var Module = fx.Provide(New)

// Params defines the dependencies of this module.
type Params struct {
	fx.In

	ProtoReflectionMetas []reflection.ServerMeta `group:"yarpcfx"`
}

// Result defines the output of this module.
type Result struct {
	fx.Out

	Procedures []yarpc.TransportProcedure `group:"yarpcfx"`
}

// New creates 'yarpc.TransportProcedure's from 'reflection.ServerMeta'.
func New(p Params) (Result, error) {
	// TODO(apeatsbond): add health.YARPCProtoHealthReflectionMeta
	procedures, err := NewServer(p.ProtoReflectionMetas)
	if err != nil {
		return Result{}, fmt.Errorf("failed to construct reflection server: %v", err)
	}

	return Result{Procedures: procedures}, nil
}
