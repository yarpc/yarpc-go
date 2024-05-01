// Copyright (c) 2024 Uber Technologies, Inc.
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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/yarpc/yarpcerrors"
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
		assert.Equal(t, tt.bytes, encodeHeaders(tt.headers))

		result, err := decodeHeaders(bytes.NewReader(tt.bytes))
		if assert.NoError(t, err) {
			assert.Equal(t, headers, result)
		}
	}
}

func TestAddCallerProcedureHeader(t *testing.T) {
	for _, tt := range []struct {
		desc            string
		treq            transport.Request
		headers         map[string]string
		expectedHeaders map[string]string
	}{
		{
			desc:    "valid_callerProcedure_and_valid_header",
			treq:    transport.Request{CallerProcedure: "ABC"},
			headers: map[string]string{"header": "value"},
			expectedHeaders: map[string]string{
				CallerProcedureHeader: "ABC",
				"header":              "value",
			},
		},
		{
			desc:            "valid_callerProcedure_and_empty_header",
			treq:            transport.Request{CallerProcedure: "ABC"},
			headers:         nil,
			expectedHeaders: map[string]string{CallerProcedureHeader: "ABC"},
		},
		{
			desc:            "empty_callerProcedure_and_empty_header",
			treq:            transport.Request{},
			headers:         nil,
			expectedHeaders: nil,
		},
		{
			desc:            "empty_callerProcedure_and_valid_header",
			treq:            transport.Request{},
			headers:         map[string]string{"header": "value"},
			expectedHeaders: map[string]string{"header": "value"},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			headers := requestToTransportHeaders(&tt.treq, tt.headers)
			assert.Equal(t, tt.expectedHeaders, headers)
		})
	}
}

func TestMoveCallerProcedureToRequest(t *testing.T) {
	for _, tt := range []struct {
		desc            string
		treq            transport.Request
		headers         map[string]string
		expectedTreq    transport.Request
		expectedHeaders map[string]string
	}{
		{
			desc:            "no_callerProcedureReq_in_headers",
			treq:            transport.Request{},
			headers:         map[string]string{"header": "value"},
			expectedTreq:    transport.Request{},
			expectedHeaders: map[string]string{"header": "value"},
		},
		{
			desc: "callerProcedureReq_set_in_headers",
			treq: transport.Request{},
			headers: map[string]string{
				"header":              "value",
				CallerProcedureHeader: "ABC",
			},
			expectedTreq:    transport.Request{CallerProcedure: "ABC"},
			expectedHeaders: map[string]string{"header": "value"},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			headers := transport.HeadersFromMap(tt.headers)
			transportHeadersToRequest(&tt.treq, headers)
			assert.Equal(t, tt.expectedTreq, tt.treq)
			assert.Equal(t, transport.HeadersFromMap(tt.expectedHeaders), headers)
		})
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
		err := writeHeaders(tt.format, tt.headers, nil, func() (tchannel.ArgWriter, error) {
			return buffer, nil
		})
		require.NoError(t, err)

		// Result must match either tt.bytes or tt.orBytes.
		if !bytes.Equal(tt.bytes, buffer.Bytes()) {
			assert.Equal(t, tt.orBytes, buffer.Bytes(), "failed for %v", tt.format)
		}

		result, err := readHeaders(tt.format, func() (tchannel.ArgReader, error) {
			reader := io.NopCloser(bytes.NewReader(buffer.Bytes()))
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

func TestWriteHeaders(t *testing.T) {
	tests := []struct {
		msg string
		// the headers are serialized in an undefined order so the encoding
		// must be one of bytes or orBytes
		bytes          []byte
		orBytes        []byte
		headers        map[string]string
		tracingBaggage map[string]string
	}{
		{
			"lowercase header",
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
			nil, /* tracingBaggage */
		},
		{
			"mixed case header",
			[]byte{
				0x00, 0x02,
				0x00, 0x01, 'A', 0x00, 0x01, '1',
				0x00, 0x01, 'b', 0x00, 0x01, '2',
			},
			[]byte{
				0x00, 0x02,
				0x00, 0x01, 'b', 0x00, 0x01, '2',
				0x00, 0x01, 'A', 0x00, 0x01, '1',
			},
			map[string]string{"A": "1", "b": "2"},
			nil, /* tracingBaggage */
		},
		{
			"keys only differ by case",
			[]byte{
				0x00, 0x02,
				0x00, 0x01, 'A', 0x00, 0x01, '1',
				0x00, 0x01, 'a', 0x00, 0x01, '2',
			},
			[]byte{
				0x00, 0x02,
				0x00, 0x01, 'a', 0x00, 0x01, '2',
				0x00, 0x01, 'A', 0x00, 0x01, '1',
			},
			map[string]string{"A": "1", "a": "2"},
			nil, /* tracingBaggage */
		},
		{
			"tracing bagger header",
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
			map[string]string{"b": "2"},
			map[string]string{"a": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			buffer := newBufferArgWriter()
			err := writeHeaders(tchannel.Raw, tt.headers, tt.tracingBaggage, func() (tchannel.ArgWriter, error) {
				return buffer, nil
			})
			require.NoError(t, err)
			// Result must match either tt.bytes or tt.orBytes.
			if !bytes.Equal(tt.bytes, buffer.Bytes()) {
				assert.Equal(t, tt.orBytes, buffer.Bytes())
			}
		})
	}
}

func TestValidateServiceHeaders(t *testing.T) {
	tests := []struct {
		name            string
		requestService  string
		responseService string
		err             bool
	}{
		{
			name:            "match",
			requestService:  "service",
			responseService: "service",
		},
		{
			name: "match empty",
		},
		{
			name:           "match - no response",
			requestService: "service",
		},
		{
			name:            "no match",
			requestService:  "foo",
			responseService: "bar",
			err:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.err {
				assert.NoError(t, validateServiceName(tt.requestService, tt.responseService))

			} else {
				err := validateServiceName(tt.requestService, tt.responseService)
				require.Error(t, err)
				assert.True(t, yarpcerrors.IsInternal(err), "expected yarpc.InternalError")
			}
		})
	}
}

func TestFindReservedHeaderPrefix(t *testing.T) {
	tests := map[string]struct {
		headers  map[string]string
		expKeys  []string
		expFound bool
	}{
		"nil-headers": {},
		"no-reserved-headers": {
			headers: map[string]string{
				"any-header-1": "any-value-1",
				"any-header-2": "any-value-2",
			},
		},
		"reserved-known-headers": {
			headers: map[string]string{
				ServiceHeaderKey: "any-value",
			},
			expKeys:  []string{ServiceHeaderKey},
			expFound: true,
		},
		"reserved-prefix": {
			headers: map[string]string{
				"rpc-any":    "any-value",
				"any-header": "any-value",
			},
			expKeys:  []string{"rpc-any"},
			expFound: true,
		},
		"reserved-dollar-prefix": {
			headers: map[string]string{
				"$rpc$-any":  "any-value",
				"any-header": "any-value",
			},
			expKeys:  []string{"$rpc$-any"},
			expFound: true,
		},
		"multiple-reserved-prefix": {
			headers: map[string]string{
				"rpc-any":    "any-value",
				"$rpc$-any":  "any-value",
				"any-header": "any-value",
			},
			expKeys:  []string{"rpc-any", "$rpc$-any"},
			expFound: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			key, found := findReservedHeaderPrefix(tt.headers)
			if len(tt.expKeys) > 0 {
				assert.Contains(t, tt.expKeys, key)
			} else {
				assert.Empty(t, key)
			}
			assert.Equal(t, tt.expFound, found)
		})
	}
}

func TestValidateApplicationHeaders(t *testing.T) {
	tests := map[string]struct {
		headers           map[string]string
		enforceHeaderRule bool
		expErr            error
		expReportHeader   bool
	}{
		"no-headers-no-error": {},
		"valid-headers-no-error": {
			headers: map[string]string{
				"valid-key": "valid-value",
			},
		},
		"reserved-rpc-header-error": {
			headers: map[string]string{
				"rpc-any": "any-value",
			},
			expReportHeader: true,
		},
		"reserved-rpc-header-error-enforced-rule": {
			headers: map[string]string{
				"rpc-any": "any-value",
			},
			enforceHeaderRule: true,
			expReportHeader:   true,
			expErr:            yarpcerrors.InternalErrorf("header with rpc prefix is not allowed in request application headers (rpc-any was passed)"),
		},
		"reserved-dollad-rpc-header-error": {
			headers: map[string]string{
				"$rpc$-any": "any-value",
			},
			expReportHeader: true,
		},
		"reserved-dollad-rpc-header-error-enforced-rule": {
			headers: map[string]string{
				"$rpc$-any": "any-value",
			},
			enforceHeaderRule: true,
			expReportHeader:   true,
			expErr:            yarpcerrors.InternalErrorf("header with rpc prefix is not allowed in request application headers ($rpc$-any was passed)"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRule)

			root := metrics.New()
			m := observability.NewReserveHeaderMetrics(root.Scope(), "tchannel")

			err := validateApplicationHeaders(tt.headers, m.With("any-source", "any-dest"))
			assert.Equal(t, tt.expErr, err)

			if tt.expReportHeader {
				assertTuple(t, root.Snapshot().Counters, tuple{"tchannel_reserved_headers_error", "any-source", "any-dest", 1})
			} else {
				assertEmptyMetrics(t, root.Snapshot())
			}
		})
	}
}

func TestDeleteReservedHeaders(t *testing.T) {
	tests := map[string]struct {
		headers                  map[string]string
		enforceHeaderRule        bool
		expHeaders               map[string]string
		expReservedHeadersMetric int64
	}{
		"nil-headers": {},
		"no-reserved-headers": {
			headers: map[string]string{
				"any-header": "any-value",
			},
			expHeaders: map[string]string{
				"any-header": "any-value",
			},
		},
		"reserved-known-headers": {
			headers: map[string]string{
				ServiceHeaderKey: "any-value",
				"any-header":     "any-value",
			},
			expHeaders: map[string]string{
				"any-header": "any-value",
			},
		},
		"reserved-rpc-headers": {
			headers: map[string]string{
				"rpc-any":    "any-value",
				"any-header": "any-value",
			},
			expHeaders: map[string]string{
				"rpc-any":    "any-value",
				"any-header": "any-value",
			},
			expReservedHeadersMetric: 1,
		},
		"reserved-dollar-rpc-headers": {
			headers: map[string]string{
				"$rpc$-any":  "any-value",
				"any-header": "any-value",
			},
			expHeaders: map[string]string{
				"$rpc$-any":  "any-value",
				"any-header": "any-value",
			},
			expReservedHeadersMetric: 1,
		},
		"enforce-header-rules": {
			headers: map[string]string{
				"rpc-any":    "any-value",
				"$rpc$-any":  "any-value",
				"any-header": "any-value",
			},
			enforceHeaderRule: true,
			expHeaders: map[string]string{
				"any-header": "any-value",
			},
			expReservedHeadersMetric: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRule)

			root := metrics.New()
			m := observability.NewReserveHeaderMetrics(root.Scope(), "tchannel")

			headers := transport.HeadersFromMap(tt.headers)
			deleteReservedHeaders(headers, m.With("any-source", "any-dest"))
			assert.Equal(t, transport.HeadersFromMap(tt.expHeaders), headers)

			if tt.expReservedHeadersMetric > 0 {
				assertTuple(t, root.Snapshot().Counters, tuple{"tchannel_reserved_headers_stripped", "any-source", "any-dest", tt.expReservedHeadersMetric})
			} else {
				assertEmptyMetrics(t, root.Snapshot())
			}
		})

	}
}

func switchEnforceHeaderRules(t *testing.T, cond bool) {
	if !cond {
		return
	}

	enforceHeaderRules = true
	t.Cleanup(func() {
		enforceHeaderRules = false
	})
}

type tuple struct {
	name, tag1, tag2 string
	value            int64
}

func assertTuple(t *testing.T, snapshots []metrics.Snapshot, expected tuple) {
	assertTuples(t, snapshots, []tuple{expected})
}

func assertTuples(t *testing.T, snapshots []metrics.Snapshot, expected []tuple) {
	actual := make([]tuple, 0, len(snapshots))

	for _, c := range snapshots {
		actual = append(actual, tuple{c.Name, c.Tags["source"], c.Tags["dest"], c.Value})
	}

	assert.ElementsMatch(t, expected, actual)
}

func assertEmptyMetrics(t *testing.T, snapshot *metrics.RootSnapshot) {
	assert.Empty(t, snapshot.Counters)
	assert.Empty(t, snapshot.Gauges)
	assert.Empty(t, snapshot.Histograms)
}
