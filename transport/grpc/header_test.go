package grpc

import (
	"testing"

	"go.uber.org/yarpc/transport"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestHeaderMapperToMetadata(t *testing.T) {
	type testStruct struct {
		mapper       headerMapper
		inputHeaders transport.Headers
		inputMD      metadata.MD
		expectedMD   metadata.MD
	}
	tests := []testStruct{
		func() (s testStruct) {
			prefix := "test-"
			s.mapper = headerMapper{prefix}
			s.inputHeaders = transport.HeadersFromMap(map[string]string{
				"key1": "value1",
				"key2": "value2",
			})
			s.inputMD = metadata.New(map[string]string{
				"key": "value",
			})
			s.expectedMD = metadata.New(map[string]string{
				"key":           "value",
				prefix + "key1": "value1",
				prefix + "key2": "value2",
			})
			return
		}(),
		func() (s testStruct) {
			prefix := "test-"
			s.mapper = headerMapper{prefix}
			s.inputHeaders = transport.HeadersFromMap(map[string]string{
				"key1": "value1",
				"key2": "value2",
			})
			s.inputMD = nil
			s.expectedMD = metadata.New(map[string]string{
				prefix + "key1": "value1",
				prefix + "key2": "value2",
			})
			return
		}(),
	}

	for _, tt := range tests {
		md := tt.mapper.toMetadata(tt.inputHeaders, tt.inputMD)

		assert.Equal(t, tt.expectedMD, md)
	}
}

func TestHeaderMapperFromMetadata(t *testing.T) {
	type testStruct struct {
		mapper          headerMapper
		inputMD         metadata.MD
		inputHeaders    transport.Headers
		expectedHeaders transport.Headers
	}
	tests := []testStruct{
		func() (s testStruct) {
			prefix := "test-"
			s.mapper = headerMapper{prefix}
			s.inputMD = metadata.New(map[string]string{
				"key10":         "value10",
				prefix + "key1": "value1",
				prefix + "key2": "value2",
			})
			s.inputHeaders = transport.HeadersFromMap(map[string]string{
				"key": "value",
			})
			s.expectedHeaders = transport.HeadersFromMap(map[string]string{
				"key":  "value",
				"key1": "value1",
				"key2": "value2",
			})
			return
		}(),
		func() (s testStruct) {
			prefix := "test-"
			s.mapper = headerMapper{prefix}
			s.inputMD = metadata.New(map[string]string{
				"key10":         "value10",
				prefix + "key1": "value1",
				prefix + "key2": "value2",
			})
			s.inputHeaders = transport.Headers{}
			s.expectedHeaders = transport.HeadersFromMap(map[string]string{
				"key1": "value1",
				"key2": "value2",
			})
			return
		}(),
	}

	for _, tt := range tests {
		headers := tt.mapper.fromMetadata(tt.inputMD, tt.inputHeaders)

		assert.Equal(t, tt.expectedHeaders, headers)
	}
}

func TestHeadersAdd(t *testing.T) {
	type testStruct struct {
		inputHeaders    headers
		addKeys         []string
		addValues       []string
		expectedHeaders headers
	}
	tests := []testStruct{
		{
			inputHeaders: map[string][]string{"key": {"value"}},
			addKeys:      []string{"key1", "key2"},
			addValues:    []string{"value1", "value2"},
			expectedHeaders: map[string][]string{
				"key":  {"value"},
				"key1": {"value1"},
				"key2": {"value2"},
			},
		},
		{
			inputHeaders: map[string][]string{"key": {"value"}},
			addKeys:      []string{"KEY1", "KEY2"},
			addValues:    []string{"VALUE1", "VALUE2"},
			expectedHeaders: map[string][]string{
				"key":  {"value"},
				"key1": {"VALUE1"},
				"key2": {"VALUE2"},
			},
		},
		{
			inputHeaders: map[string][]string{"key": {"value"}},
			addKeys:      []string{"key"},
			addValues:    []string{"value2"},
			expectedHeaders: map[string][]string{
				"key": {"value", "value2"},
			},
		},
	}

	for _, tt := range tests {
		headers := tt.inputHeaders
		for i := range tt.addKeys {
			headers.add(tt.addKeys[i], tt.addValues[i])
		}
		assert.Equal(t, tt.expectedHeaders, headers)
	}
}

func TestHeadersDel(t *testing.T) {
	type testStruct struct {
		inputHeaders    headers
		delKeys         []string
		expectedHeaders headers
	}
	tests := []testStruct{
		{
			inputHeaders: map[string][]string{
				"key":  {"value"},
				"key1": {"value1"},
				"key2": {"value2"},
			},
			delKeys: []string{"key1", "key2"},
			expectedHeaders: map[string][]string{
				"key": {"value"},
			},
		},
		{
			inputHeaders: map[string][]string{
				"key":  {"value"},
				"key1": {"value1"},
				"key2": {"value2"},
			},
			delKeys: []string{"KEY1", "KEY2"},
			expectedHeaders: map[string][]string{
				"key": {"value"},
			},
		},
	}

	for _, tt := range tests {
		headers := tt.inputHeaders
		for _, key := range tt.delKeys {
			headers.del(key)
		}
		assert.Equal(t, tt.expectedHeaders, headers)
	}
}

func TestHeadersGet(t *testing.T) {
	type testStruct struct {
		inputHeaders   headers
		keys           []string
		expectedValues []string
	}
	tests := []testStruct{
		{
			inputHeaders: map[string][]string{
				"key":  {"value"},
				"key1": {"value1"},
				"key2": {"value2"},
			},
			keys:           []string{"key1", "key2"},
			expectedValues: []string{"value1", "value2"},
		},
		{
			inputHeaders: map[string][]string{
				"key":  {"value"},
				"key1": {"value1"},
				"key2": {"value2"},
			},
			keys:           []string{"KEY1"},
			expectedValues: []string{"value1"},
		},
		{
			inputHeaders:   nil,
			keys:           []string{"key1"},
			expectedValues: []string{""},
		},
		{
			inputHeaders: map[string][]string{
				"key": {"value"},
			},
			keys:           []string{"key1"},
			expectedValues: []string{""},
		},
		{
			inputHeaders:   map[string][]string{},
			keys:           []string{"key1"},
			expectedValues: []string{""},
		},
	}

	for _, tt := range tests {
		headers := tt.inputHeaders
		for i := range tt.keys {
			value := headers.get(tt.keys[i])
			assert.Equal(t, tt.expectedValues[i], value)
		}
		assert.Equal(t, tt.inputHeaders, headers) // Ensure there were no mutations on get commands
	}
}

func TestHeadersSet(t *testing.T) {
	type testStruct struct {
		inputHeaders    headers
		setKeys         []string
		setValues       []string
		expectedHeaders headers
	}
	tests := []testStruct{
		{
			inputHeaders: map[string][]string{
				"key": {"value"},
			},
			setKeys:   []string{"key1", "key2"},
			setValues: []string{"value1", "value2"},
			expectedHeaders: map[string][]string{
				"key":  {"value"},
				"key1": {"value1"},
				"key2": {"value2"},
			},
		},
		{
			inputHeaders: map[string][]string{
				"key": {"value", "value1"},
			},
			setKeys:   []string{"key"},
			setValues: []string{"value4"},
			expectedHeaders: map[string][]string{
				"key": {"value4"},
			},
		},
		{
			inputHeaders: map[string][]string{
				"key": {"value", "value1"},
			},
			setKeys:   []string{"KEY"},
			setValues: []string{"value4"},
			expectedHeaders: map[string][]string{
				"key": {"value4"},
			},
		},
	}

	for _, tt := range tests {
		headers := tt.inputHeaders
		for i := range tt.setKeys {
			headers.set(tt.setKeys[i], tt.setValues[i])
		}
		assert.Equal(t, tt.expectedHeaders, headers)
	}
}
