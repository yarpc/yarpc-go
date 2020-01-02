// Copyright (c) 2020 Uber Technologies, Inc.
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

package yarpcmeta

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
)

func TestProcedures(t *testing.T) {
	disp := yarpc.NewDispatcher(yarpc.Config{
		Name: "myservice",
	})
	ms := &service{disp}

	r, err := ms.procs(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "myservice", r.Service)
	found := false
	for _, p := range r.Procedures {
		if p.Name == "myprocedure" {
			found = true
			break
		}
	}
	assert.False(t, found)

	disp.Register(json.Procedure("myprocedure",
		func(context.Context, interface{}) (interface{}, error) {
			return nil, nil
		}))

	r, err = ms.procs(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "myservice", r.Service)
	found = false
	for _, p := range r.Procedures {
		if p.Name == "myprocedure" {
			found = true
			break
		}
	}
	assert.True(t, found)
}
