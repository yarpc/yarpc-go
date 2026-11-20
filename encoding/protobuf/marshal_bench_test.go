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

package protobuf

import (
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

// BenchmarkMarshal measures marshaling large messages of different shapes.
//
// Run with:
//
//	go test -bench=BenchmarkMarshal -benchmem ./encoding/protobuf
func BenchmarkMarshal(b *testing.B) {
	shapes := []struct {
		name string
		make func(size int) proto.Message
	}{
		{
			name: "bytes",
			make: func(size int) proto.Message { return &types.BytesValue{Value: make([]byte, size)} },
		},
		{
			name: "struct",
			make: newStructList,
		},
	}
	sizes := []struct {
		name string
		size int
	}{
		{"512KB", 512 * 1024},
		{"1MB", 1024 * 1024},
		{"2MB", 2 * 1024 * 1024},
		{"8MB", 8 * 1024 * 1024},
	}

	for _, shape := range shapes {
		for _, size := range sizes {
			msg := shape.make(size.size)
			b.Run(shape.name+"/"+size.name, func(b *testing.B) {
				b.SetBytes(int64(proto.Size(msg)))
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, cleanup, err := marshal(Encoding, msg, nil)
					if err != nil {
						b.Fatal(err)
					}
					cleanup()
				}
			})
		}
	}
}

// newStructList returns a list of small structs that marshals to at least size
// bytes.
func newStructList(size int) proto.Message {
	newValue := func(i int) *types.Value {
		return &types.Value{Kind: &types.Value_StructValue{
			StructValue: &types.Struct{Fields: map[string]*types.Value{
				"id":        {Kind: &types.Value_NumberValue{NumberValue: float64(i)}},
				"type":      {Kind: &types.Value_StringValue{StringValue: "resource.item.updated"}},
				"timestamp": {Kind: &types.Value_NumberValue{NumberValue: 1.7e9 + float64(i)}},
				"payload":   {Kind: &types.Value_StringValue{StringValue: strings.Repeat("x", 128)}},
			}},
		}}
	}

	n := size / proto.Size(&types.ListValue{Values: []*types.Value{newValue(0)}})
	list := &types.ListValue{}
	for i := 0; i < n || proto.Size(list) < size; i++ {
		list.Values = append(list.Values, newValue(i))
	}
	return list
}
