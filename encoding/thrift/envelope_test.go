// Copyright (c) 2026 Uber Technologies, Inc.
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
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
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
		proto := disableEnvelopingProtocol{binary.Default, wire.Reply}
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

func TestDisableEnveloperNoWireRead(t *testing.T) {
	cont := "some buffered contents"
	buf := strings.NewReader(cont)
	evnw := disableEnvelopingNoWireProtocol{Protocol: binary.Default, Type: wire.Call}
	sr := evnw.Reader(buf)
	evh, err := sr.ReadEnvelopeBegin()
	require.NoError(t, err)
	assert.Equal(t, wire.Call, evh.Type)

	err = sr.ReadEnvelopeEnd()
	require.NoError(t, err)

	rem, err := io.ReadAll(buf)
	require.NoError(t, err)
	assert.Equal(t, cont, string(rem), "readenvelope is not supposed to read anything from the buffer")
}

func TestDisableEnveloperNoWireWrite(t *testing.T) {
	buf := bytes.Buffer{}
	evnw := disableEnvelopingNoWireProtocol{Protocol: binary.Default, Type: wire.OneWay}
	sw := evnw.Writer(&buf)

	err := sw.WriteEnvelopeBegin(stream.EnvelopeHeader{Name: "foo", Type: wire.Exception})
	require.NoError(t, err)

	err = sw.WriteEnvelopeEnd()
	assert.NoError(t, err)
	assert.Zero(t, buf.Len(), "writeenvelope is not supposed to write to the buffer")
}

// generate generates a random value into the given pointer.
//
//	var i int
//	generate(&i, rand)
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
