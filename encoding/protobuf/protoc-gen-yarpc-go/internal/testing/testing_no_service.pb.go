// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing_no_service.proto

// Copyright (c) 2019 Uber Technologies, Inc.
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

package testing

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

import strings "strings"
import reflect "reflect"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type Foo struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Foo) Reset()      { *m = Foo{} }
func (*Foo) ProtoMessage() {}
func (*Foo) Descriptor() ([]byte, []int) {
	return fileDescriptor_testing_no_service_cbcc38bc53ab705d, []int{0}
}
func (m *Foo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Foo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Foo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *Foo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Foo.Merge(dst, src)
}
func (m *Foo) XXX_Size() int {
	return m.Size()
}
func (m *Foo) XXX_DiscardUnknown() {
	xxx_messageInfo_Foo.DiscardUnknown(m)
}

var xxx_messageInfo_Foo proto.InternalMessageInfo

func (m *Foo) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func init() {
	proto.RegisterType((*Foo)(nil), "uber.yarpc.encoding.protobuf.protocgenyarpcgo.internal.testing.Foo")
}
func (this *Foo) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Foo)
	if !ok {
		that2, ok := that.(Foo)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Id != that1.Id {
		return false
	}
	return true
}
func (this *Foo) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&testing.Foo{")
	s = append(s, "Id: "+fmt.Sprintf("%#v", this.Id)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringTestingNoService(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *Foo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Foo) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Id) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintTestingNoService(dAtA, i, uint64(len(m.Id)))
		i += copy(dAtA[i:], m.Id)
	}
	return i, nil
}

func encodeVarintTestingNoService(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *Foo) Size() (n int) {
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovTestingNoService(uint64(l))
	}
	return n
}

func sovTestingNoService(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozTestingNoService(x uint64) (n int) {
	return sovTestingNoService(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Foo) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Foo{`,
		`Id:` + fmt.Sprintf("%v", this.Id) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringTestingNoService(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Foo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTestingNoService
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Foo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Foo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTestingNoService
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTestingNoService
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTestingNoService(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthTestingNoService
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipTestingNoService(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTestingNoService
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTestingNoService
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTestingNoService
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthTestingNoService
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowTestingNoService
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipTestingNoService(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthTestingNoService = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTestingNoService   = fmt.Errorf("proto: integer overflow")
)

func init() {
	proto.RegisterFile("encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing_no_service.proto", fileDescriptor_testing_no_service_cbcc38bc53ab705d)
}

var fileDescriptor_testing_no_service_cbcc38bc53ab705d = []byte{
	// 203 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xf2, 0x4f, 0xcd, 0x4b, 0xce,
	0x4f, 0xc9, 0xcc, 0x4b, 0xd7, 0x2f, 0x28, 0xca, 0x2f, 0xc9, 0x4f, 0x2a, 0x4d, 0x83, 0x30, 0x92,
	0x75, 0xd3, 0x53, 0xf3, 0x74, 0x2b, 0x13, 0x8b, 0x0a, 0x92, 0x75, 0xd3, 0xf3, 0xf5, 0x33, 0xf3,
	0x4a, 0x52, 0x8b, 0xf2, 0x12, 0x73, 0xf4, 0x4b, 0x52, 0x8b, 0x4b, 0x40, 0xaa, 0xa1, 0x74, 0x7c,
	0x5e, 0x7e, 0x7c, 0x71, 0x6a, 0x51, 0x59, 0x66, 0x72, 0xaa, 0x1e, 0x58, 0x9f, 0x90, 0x5d, 0x69,
	0x52, 0x6a, 0x91, 0x1e, 0x58, 0xa3, 0x1e, 0xcc, 0x6c, 0x3d, 0x98, 0xd9, 0x10, 0x46, 0x72, 0x7a,
	0x6a, 0x1e, 0x58, 0x41, 0x7a, 0xbe, 0x1e, 0xcc, 0x60, 0x3d, 0xa8, 0x81, 0x4a, 0xa2, 0x5c, 0xcc,
	0x6e, 0xf9, 0xf9, 0x42, 0x7c, 0x5c, 0x4c, 0x99, 0x29, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41,
	0x4c, 0x99, 0x29, 0x4e, 0xa6, 0x17, 0x1e, 0xca, 0x31, 0xdc, 0x78, 0x28, 0xc7, 0xf0, 0xe1, 0xa1,
	0x1c, 0x63, 0xc3, 0x23, 0x39, 0xc6, 0x15, 0x8f, 0xe4, 0x18, 0x4f, 0x3c, 0x92, 0x63, 0xbc, 0xf0,
	0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x17, 0x8f, 0xe4, 0x18, 0x3e, 0x3c, 0x92, 0x63, 0x9c,
	0xf0, 0x58, 0x8e, 0x21, 0x8a, 0x1d, 0x6a, 0x5a, 0x12, 0x1b, 0xd8, 0x42, 0x63, 0x40, 0x00, 0x00,
	0x00, 0xff, 0xff, 0x96, 0xca, 0x67, 0x73, 0xe7, 0x00, 0x00, 0x00,
}
