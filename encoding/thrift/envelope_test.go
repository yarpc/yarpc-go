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

package thrift

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

func TestDisableEnveloperEncode(t *testing.T) {
	rand := rand.New(rand.NewSource(time.Now().Unix()))

	tests := []struct {
		value wire.Value
		want  []byte
	}{
		{
			wire.NewValueStruct(wire.Struct{Fields: []wire.Field{}}),
			[]byte{0x00},
		},
		{
			wire.NewValueStruct(wire.Struct{Fields: []wire.Field{
				{ID: 1, Value: wire.NewValueI32(42)},
			}}),
			[]byte{
				0x08, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x2a,
				0x00,
			},
		},
	}

	for _, tt := range tests {
		e := wire.Envelope{Value: tt.value, Type: wire.Call}
		generate(&e.Name, rand)
		generate(&e.SeqID, rand)

		var buffer bytes.Buffer
		proto := disableEnvelopingProtocol{protocol.Binary, wire.Reply}
		if !assert.NoError(t, proto.EncodeEnveloped(e, &buffer)) {
			continue
		}

		assert.Equal(t, tt.want, buffer.Bytes())

		gotE, err := proto.DecodeEnveloped(bytes.NewReader(tt.want))
		if !assert.NoError(t, err) {
			continue
		}

		assert.Equal(t, wire.Reply, gotE.Type)
		assert.True(t, wire.ValuesAreEqual(tt.value, gotE.Value))
	}
}

// generate generates a random value into the given pointer.
//
// 	var i int
// 	generate(&i, rand)
//
// If the type implements the quick.Generator interface, that is used.
func generate(v interface{}, r *rand.Rand) {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("%v is not a pointer type", t))
	}

	out, ok := quick.Value(t.Elem(), r)
	if !ok {
		panic(fmt.Sprintf("could not generate a value for %v", t))
	}

	reflect.ValueOf(v).Elem().Set(out)
}
