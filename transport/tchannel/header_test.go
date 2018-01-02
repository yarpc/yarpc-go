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

package tchannel

import (
	"bytes"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
)

func TestEncodeAndDecodeHeaders(t *testing.T) {
	tests := []struct {
		bytes   []byte
		headers map[string]string
	}{
		{[]byte{0x00, 0x00}, nil},
		{
			[]byte{
				0x00, 0x01, // 1 header

				0x00, 0x05, // length = 5
				'h', 'e', 'l', 'l', 'o',

				0x00, 0x05, // lengtth = 5
				'w', 'o', 'r', 'l', 'd',
			},
			map[string]string{"hello": "world"},
		},
	}

	for _, tt := range tests {
		headers := transport.HeadersFromMap(tt.headers)
		assert.Equal(t, tt.bytes, encodeHeaders(headers))

		result, err := decodeHeaders(bytes.NewReader(tt.bytes))
		if assert.NoError(t, err) {
			assert.Equal(t, headers, result)
		}
	}
}

func TestDecodeHeaderErrors(t *testing.T) {
	tests := [][]byte{
		{0x00, 0x01},
		{
			0x00, 0x01,
			0x00, 0x02, 'a',
			0x00, 0x01, 'b',
		},
	}

	for _, tt := range tests {
		_, err := decodeHeaders(bytes.NewReader(tt))
		assert.Error(t, err)
	}
}

func TestReadAndWriteHeaders(t *testing.T) {
	tests := []struct {
		format tchannel.Format

		// the headers are serialized in an undefined order so the encoding
		// must be one of the following
		bytes   []byte
		orBytes []byte

		headers map[string]string
	}{
		{
			tchannel.Raw,
			[]byte{
				0x00, 0x02,
				0x00, 0x01, 'a', 0x00, 0x01, '1',
				0x00, 0x01, 'b', 0x00, 0x01, '2',
			},
			[]byte{
				0x00, 0x02,
				0x00, 0x01, 'b', 0x00, 0x01, '2',
				0x00, 0x01, 'a', 0x00, 0x01, '1',
			},
			map[string]string{"a": "1", "b": "2"},
		},
		{
			tchannel.JSON,
			[]byte(`{"a":"1","b":"2"}` + "\n"),
			[]byte(`{"b":"2","a":"1"}` + "\n"),
			map[string]string{"a": "1", "b": "2"},
		},
		{
			tchannel.Thrift,
			[]byte{
				0x00, 0x02,
				0x00, 0x01, 'a', 0x00, 0x01, '1',
				0x00, 0x01, 'b', 0x00, 0x01, '2',
			},
			[]byte{
				0x00, 0x02,
				0x00, 0x01, 'b', 0x00, 0x01, '2',
				0x00, 0x01, 'a', 0x00, 0x01, '1',
			},
			map[string]string{"a": "1", "b": "2"},
		},
	}

	for _, tt := range tests {
		headers := transport.HeadersFromMap(tt.headers)

		buffer := newBufferArgWriter()
		err := writeHeaders(tt.format, headers, func() (tchannel.ArgWriter, error) {
			return buffer, nil
		})
		require.NoError(t, err)

		// Result must match either tt.bytes or tt.orBytes.
		if !bytes.Equal(tt.bytes, buffer.Bytes()) {
			assert.Equal(t, tt.orBytes, buffer.Bytes(), "failed for %v", tt.format)
		}

		result, err := readHeaders(tt.format, func() (tchannel.ArgReader, error) {
			reader := ioutil.NopCloser(bytes.NewReader(buffer.Bytes()))
			return tchannel.ArgReader(reader), nil
		})
		require.NoError(t, err)
		assert.Equal(t, headers, result, "failed for %v", tt.format)
	}
}

func TestReadHeadersFailure(t *testing.T) {
	_, err := readHeaders(tchannel.Raw, func() (tchannel.ArgReader, error) {
		return nil, errors.New("great sadness")
	})
	require.Error(t, err)
}
