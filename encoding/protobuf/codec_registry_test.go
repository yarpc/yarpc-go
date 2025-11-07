package protobuf

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/mem"
)

func TestCodecRegistry(t *testing.T) {
	// Test codec registration and retrieval
	t.Run("codecRegistration", func(t *testing.T) {
		// Create a mock codec
		mockCodec := &MockYARPCCodec{name: "test-codec"}

		// Register codec
		RegisterCodec(mockCodec)

		// Test retrieval by codec name
		retrieved := getCodecForEncoding("test-codec", nil)
		assert.Equal(t, mockCodec, retrieved, "Should return registered codec")

		// Test fallback for unknown encoding
		unknown := getCodecForEncoding("unknown-encoding", nil)
		assert.Nil(t, unknown, "Should return nil for unknown encoding")
	})

	// Test public API
	t.Run("publicAPI", func(t *testing.T) {
		mockCodec := &MockYARPCCodec{name: "public-test"}

		// Register codec
		RegisterCodec(mockCodec)

		// Test public GetCodecForEncoding function
		retrieved := GetCodecForEncoding("public-test")
		assert.Equal(t, mockCodec, retrieved, "Public API should return registered codec")
	})

	// Test thread safety (basic check)
	t.Run("concurrentAccess", func(t *testing.T) {
		codec1 := &MockYARPCCodec{name: "concurrent-1"}
		codec2 := &MockYARPCCodec{name: "concurrent-2"}

		// Register from multiple goroutines
		done := make(chan bool, 2)

		go func() {
			RegisterCodec(codec1)
			done <- true
		}()

		go func() {
			RegisterCodec(codec2)
			done <- true
		}()

		// Wait for both registrations
		<-done
		<-done

		// Verify both are registered correctly
		assert.Equal(t, codec1, getCodecForEncoding("concurrent-1", nil))
		assert.Equal(t, codec2, getCodecForEncoding("concurrent-2", nil))
	})

	// Test codec interface compliance
	t.Run("codecInterface", func(t *testing.T) {
		mockCodec := &MockYARPCCodec{name: "interface-test"}

		// Test Marshal
		data, err := mockCodec.Marshal([]byte("test-data"))
		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Test Unmarshal
		var result []byte
		bufSlice := mem.BufferSlice{mem.SliceBuffer([]byte("unmarshal-test"))}
		err = mockCodec.Unmarshal(bufSlice, &result)
		assert.NoError(t, err)
		assert.Equal(t, []byte("unmarshal-test"), result)

		// Test Name
		assert.Equal(t, "interface-test", mockCodec.Name())
	})
}

// MockYARPCCodec implements encoding.CodecV2 for testing
type MockYARPCCodec struct {
	name string
}

func (m *MockYARPCCodec) Marshal(v any) (mem.BufferSlice, error) {
	switch value := v.(type) {
	case []byte:
		return mem.BufferSlice{mem.SliceBuffer(value)}, nil
	default:
		return nil, fmt.Errorf("expected []byte but got %T", v)
	}
}

func (m *MockYARPCCodec) Unmarshal(data mem.BufferSlice, v any) error {
	switch value := v.(type) {
	case *[]byte:
		*value = data.Materialize()
		return nil
	default:
		return fmt.Errorf("expected *[]byte but got %T", v)
	}
}

func (m *MockYARPCCodec) Name() string {
	return m.name
}
