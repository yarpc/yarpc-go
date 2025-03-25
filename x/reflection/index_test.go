package reflection

import (
	"bytes"
	"compress/gzip"
	"embed"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/encoding/protobuf/reflection"
)

func gzipBytes(t *testing.T, b []byte) []byte {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	require.NoError(t, err)
	_, err = w.Write(b)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

//go:embed testdata/*
//go:embed testdata/otherroot/*
var testdata embed.FS

func loadTestFileDescriptor(t *testing.T, fileName string) []byte {
	b, err := testdata.ReadFile("testdata/" + fileName)
	require.NoError(t, err)
	fdSet := &descriptor.FileDescriptorSet{}
	err = proto.Unmarshal(b, fdSet)
	require.NoError(t, err)
	require.Equal(t, 1, len(fdSet.File))
	fd, err := proto.Marshal(fdSet.File[0])
	require.NoError(t, err)
	return fd
}

func loadTestFileDescriptorCompressed(t *testing.T, fileName string) []byte {
	return gzipBytes(t, loadTestFileDescriptor(t, fileName))
}

func TestIndexReflectionMetas(t *testing.T) {
	mainFileDescriptor := loadTestFileDescriptor(t, "main.proto.bin")
	mainFileDescriptorCompressed := loadTestFileDescriptorCompressed(t, "main.proto.bin")
	otherFileDescriptor := loadTestFileDescriptor(t, "other.proto.bin")
	otherFileDescriptorCompressed := loadTestFileDescriptorCompressed(t, "other.proto.bin")
	tests := []struct {
		name         string
		args         []reflection.ServerMeta
		wantServices []string
		wantIndex    fileDescriptorIndex
		wantErr      string
	}{
		{
			name:    "invalid bytes",
			args:    []reflection.ServerMeta{{FileDescriptors: [][]byte{{0x00}}}},
			wantErr: "failed to decompress enc: bad gzipped descriptor: unexpected EOF",
		},
		{
			name:    "not a fd",
			args:    []reflection.ServerMeta{{FileDescriptors: [][]byte{gzipBytes(t, []byte{0x00})}}},
			wantErr: "bad descriptor: proto: descriptor.FileDescriptorProto: illegal tag 0 (wire type 0)",
		},
		{
			name: "conflicting filename registration",
			args: []reflection.ServerMeta{
				{ServiceName: "someService", FileDescriptors: [][]byte{mainFileDescriptorCompressed}},
				{ServiceName: "otherServic", FileDescriptors: [][]byte{loadTestFileDescriptorCompressed(t, "otherroot/main.proto.bin")}},
			},
			wantErr: `raw filedescriptor for "main.proto" provided by service "otherServic" do not match bytes provided by service "someService"`,
		},
		{
			name: "conflicting extension",
			args: []reflection.ServerMeta{
				{ServiceName: "someService", FileDescriptors: [][]byte{mainFileDescriptorCompressed}},
				{ServiceName: "otherServic", FileDescriptors: [][]byte{loadTestFileDescriptorCompressed(t, "conflict_extension.proto.bin")}},
			},
			wantErr: `extension name and number already indexed: "attribute" 808080`,
		},
		{
			name: "conflicting message registration",
			args: []reflection.ServerMeta{
				{ServiceName: "someService", FileDescriptors: [][]byte{mainFileDescriptorCompressed}},
				{ServiceName: "otherServic", FileDescriptors: [][]byte{loadTestFileDescriptorCompressed(t, "conflict_message.proto.bin")}},
			},
			wantErr: `symbol name already indexed: "Foo"`,
		},
		{
			name: "conflicting message extension registration",
			args: []reflection.ServerMeta{
				{ServiceName: "otherServic", FileDescriptors: [][]byte{loadTestFileDescriptorCompressed(t, "conflict_message_extension.proto.bin")}},
				{ServiceName: "someService", FileDescriptors: [][]byte{mainFileDescriptorCompressed}},
			},
			wantErr: `extension name and number already indexed: "Foo.isAFoo" 424242`,
		},
		{
			name: "conflicting message nest registration",
			args: []reflection.ServerMeta{
				{ServiceName: "otherServic", FileDescriptors: [][]byte{loadTestFileDescriptorCompressed(t, "conflict_message_nest.proto.bin")}},
				{ServiceName: "someService", FileDescriptors: [][]byte{mainFileDescriptorCompressed}},
			},
			wantErr: `symbol name already indexed: "Foo.NestedFoo"`,
		},
		{
			name: "conflicting service method registration",
			args: []reflection.ServerMeta{
				{ServiceName: "otherServic", FileDescriptors: [][]byte{loadTestFileDescriptorCompressed(t, "conflict_service_method.proto.bin")}},
				{ServiceName: "someService", FileDescriptors: [][]byte{mainFileDescriptorCompressed}},
			},
			wantErr: `symbol name already indexed: "Bar.Baz"`,
		},
		{
			name: "conflicting service registration",
			args: []reflection.ServerMeta{
				{ServiceName: "otherServic", FileDescriptors: [][]byte{loadTestFileDescriptorCompressed(t, "conflict_service.proto.bin")}},
				{ServiceName: "someService", FileDescriptors: [][]byte{mainFileDescriptorCompressed}},
			},
			wantErr: `symbol name already indexed: "Bar"`,
		},
		{
			name:         "none",
			wantServices: []string{},
			wantIndex: &descriptorIndex{
				descriptorsBySymbol:             map[string][]byte{},
				descriptorsByFile:               map[string][]byte{},
				descriptorsByExtensionAndNumber: map[string]map[int32][]byte{},
			},
		},
		{
			name: "success",
			args: []reflection.ServerMeta{
				{ServiceName: "Bar", FileDescriptors: [][]byte{mainFileDescriptorCompressed, otherFileDescriptorCompressed}},
			},
			wantServices: []string{"Bar"},
			wantIndex: &descriptorIndex{
				descriptorsBySymbol: map[string][]byte{
					"Bar":           mainFileDescriptor,
					"Bar.Baz":       mainFileDescriptor,
					"Foo":           mainFileDescriptor,
					"Foo.NestedFoo": mainFileDescriptor,
				},
				descriptorsByFile: map[string][]byte{
					"main.proto":  mainFileDescriptor,
					"other.proto": otherFileDescriptor,
				},
				descriptorsByExtensionAndNumber: map[string]map[int32][]byte{
					"attribute":  {808080: mainFileDescriptor, 808081: otherFileDescriptor},
					"Foo.isAFoo": {424242: mainFileDescriptor},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			services, index, err := indexReflectionMetas(tt.args)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if !reflect.DeepEqual(services, tt.wantServices) {
				t.Errorf("indexReflectionMetas() got = %v, want %v", services, tt.wantServices)
			}
			dIndex, ok := index.(*descriptorIndex)
			assert.True(t, ok)
			assert.NotNil(t, dIndex)
			if !reflect.DeepEqual(dIndex, tt.wantIndex) {
				t.Errorf("indexReflectionMetas() index = %v, want %v", index, tt.wantIndex)
			}
		})
	}
}
