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
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"go.uber.org/yarpc/v2/yarpcprotobuf/reflection"
)

type fileDescriptorIndex interface {
	descriptorForFile(string) ([]byte, error)
	descriptorForSymbol(string) ([]byte, error)
	descriptorForExtensionAndNumber(string, int32) ([]byte, error)
	allExtensionNumbersForTypeName(string) ([]int32, error)
}

type namedFileDescriptor struct {
	service             string
	fileDescriptor      *descriptor.FileDescriptorProto
	fileDescriptorBytes []byte
}

func indexReflectionMetas(metas []reflection.ServerMeta) ([]string, fileDescriptorIndex, error) {
	serviceNames := make([]string, 0, len(metas))
	filesToProcess := map[string]namedFileDescriptor{}
	for _, m := range metas {
		serviceNames = append(serviceNames, m.ServiceName)
		for _, fdbytes := range m.FileDescriptors {
			fd, fdBytes, err := decodeFileDesc(fdbytes)
			if err != nil {
				return nil, nil, err
			}
			fileName := fd.GetName()
			// Each reflection.ServerMeta contains a full transitive closure of the serialized proto files
			// required for the reflection server. Validate that the files with the same name from possibly
			// different meta's contain the same contents, error if they don't as there is no reasonable way
			// to recover from this inconsistent state.
			if h, ok := filesToProcess[fileName]; ok {
				if !bytes.Equal(fdBytes, h.fileDescriptorBytes) {
					return nil, nil, fmt.Errorf("raw filedescriptor for %q provided by service %q do not match bytes provided by service %q", fileName, m.ServiceName, h.service)
				}
			} else {
				filesToProcess[fileName] = namedFileDescriptor{
					service:             m.ServiceName,
					fileDescriptor:      fd,
					fileDescriptorBytes: fdBytes,
				}
			}
		}
	}
	f := &descriptorIndex{
		descriptorsBySymbol:             map[string][]byte{},
		descriptorsByFile:               map[string][]byte{},
		descriptorsByExtensionAndNumber: map[string]map[int32][]byte{},
	}
	for _, v := range filesToProcess {
		err := f.indexFile(v.fileDescriptorBytes, v.fileDescriptor)
		if err != nil {
			return nil, nil, err
		}
	}
	sort.Strings(serviceNames)

	return serviceNames, f, nil
}

type descriptorIndex struct {
	descriptorsBySymbol             map[string][]byte
	descriptorsByFile               map[string][]byte
	descriptorsByExtensionAndNumber map[string]map[int32][]byte
}

func (f *descriptorIndex) descriptorForFile(file string) ([]byte, error) {
	fd, ok := f.descriptorsByFile[file]
	if !ok {
		return nil, fmt.Errorf("could not find descriptor for file %q", file)
	}
	return fd, nil
}

func (f *descriptorIndex) descriptorForSymbol(symbol string) ([]byte, error) {
	fd, ok := f.descriptorsBySymbol[symbol]
	if !ok {
		return nil, fmt.Errorf("could not find descriptor for symbol %q", symbol)
	}
	return fd, nil
}

func (f *descriptorIndex) descriptorForExtensionAndNumber(extension string, number int32) ([]byte, error) {
	extNums, ok := f.descriptorsByExtensionAndNumber[extension]
	if !ok {
		return nil, fmt.Errorf("could not find extension type %q", extension)
	}
	fd, ok := extNums[number]
	if !ok {
		return nil, fmt.Errorf("could not find extension number %d on type %q", number, extension)
	}
	return fd, nil
}

func (f *descriptorIndex) allExtensionNumbersForTypeName(extension string) ([]int32, error) {
	extMap, ok := f.descriptorsByExtensionAndNumber[extension]
	if !ok {
		return nil, fmt.Errorf("could not find extension type %q", extension)
	}
	extNums := make([]int32, 0, len(extMap))
	for n := range extMap {
		extNums = append(extNums, n)
	}
	return extNums, nil
}

func (f *descriptorIndex) indexFile(fdBytes []byte, fd *descriptor.FileDescriptorProto) error {
	filename := fd.GetName()
	if err := f.setFileDescriptorForFile(fdBytes, filename); err != nil {
		return err
	}

	prefix := fd.GetPackage()
	for _, msg := range fd.MessageType {
		if err := f.indexMessage(fdBytes, prefix, msg); err != nil {
			return err
		}
	}
	for _, ext := range fd.Extension {
		if err := f.indexExtension(fdBytes, prefix, ext); err != nil {
			return err
		}
	}
	for _, svc := range fd.Service {
		if err := f.indexService(fdBytes, prefix, svc); err != nil {
			return err
		}
	}
	return nil
}

func (f *descriptorIndex) indexService(fdBytes []byte, prefix string, svc *descriptor.ServiceDescriptorProto) error {
	svcName := fqn(prefix, svc.GetName())
	err := f.setFileDescriptorForSymbol(fdBytes, svcName)
	if err != nil {
		return err
	}

	for _, meth := range svc.Method {
		methodName := fqn(svcName, meth.GetName())
		err := f.setFileDescriptorForSymbol(fdBytes, methodName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *descriptorIndex) indexMessage(fdBytes []byte, prefix string, msg *descriptor.DescriptorProto) error {
	msgName := fqn(prefix, msg.GetName())
	err := f.setFileDescriptorForSymbol(fdBytes, msgName)
	if err != nil {
		return err
	}

	for _, nested := range msg.NestedType {
		err := f.indexMessage(fdBytes, msgName, nested)
		if err != nil {
			return err
		}
	}
	for _, ext := range msg.Extension {
		err := f.indexExtension(fdBytes, msgName, ext)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *descriptorIndex) indexExtension(fdBytes []byte, prefix string, ext *descriptor.FieldDescriptorProto) error {
	name := fqn(prefix, ext.GetName())
	number := ext.GetNumber()
	if _, ok := f.descriptorsByExtensionAndNumber[name]; !ok {
		f.descriptorsByExtensionAndNumber[name] = map[int32][]byte{}
	}
	if _, ok := f.descriptorsByExtensionAndNumber[name][number]; ok {
		return fmt.Errorf("extension name and number already indexed: %q %d", name, number)
	}
	f.descriptorsByExtensionAndNumber[name][number] = fdBytes
	return nil
}

func (f *descriptorIndex) setFileDescriptorForSymbol(fdBytes []byte, symbolName string) error {
	if _, ok := f.descriptorsBySymbol[symbolName]; ok {
		return fmt.Errorf("symbol name already indexed: %q", symbolName)
	}
	f.descriptorsBySymbol[symbolName] = fdBytes
	return nil
}

func (f *descriptorIndex) setFileDescriptorForFile(fdBytes []byte, filename string) error {
	if _, ok := f.descriptorsByFile[filename]; ok {
		return fmt.Errorf("filename already indexed %q", filename)
	}
	f.descriptorsByFile[filename] = fdBytes
	return nil
}

func fqn(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// decodeFileDesc does decompression and unmarshalling on the given
// file descriptor byte slice.
func decodeFileDesc(enc []byte) (*descriptor.FileDescriptorProto, []byte, error) {
	raw, err := decompress(enc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decompress enc: %v", err)
	}

	fd := &descriptor.FileDescriptorProto{}
	if err := proto.Unmarshal(raw, fd); err != nil {
		return nil, nil, fmt.Errorf("bad descriptor: %v", err)
	}
	return fd, raw, nil
}

// decompress does gzip decompression.
func decompress(b []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("bad gzipped descriptor: %v", err)
	}
	return ioutil.ReadAll(r)
}
