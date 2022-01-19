// Copyright (c) 2022 Uber Technologies, Inc.
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

package yarpcconfig

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/config"
)

var _typeOfEmptyStruct = reflect.TypeOf(struct{}{})

func TestCompileTransportSpec(t *testing.T) {
	type phoneCall struct{ Message string }
	type cavalry struct{ Horses int }
	type debt struct{ Amount int64 }

	tests := []struct {
		desc string
		spec TransportSpec

		supportsUnary  bool
		supportsOneway bool
		supportsStream bool

		transportInput      reflect.Type
		inboundInput        reflect.Type
		unaryOutboundInput  reflect.Type
		onewayOutboundInput reflect.Type
		streamOutboundInput reflect.Type

		wantErr []string
	}{
		{
			desc:    "missing name",
			wantErr: []string{"field Name is required"},
		},
		{
			desc:    "reserved name",
			spec:    TransportSpec{Name: "Unary"},
			wantErr: []string{`transport name cannot be "Unary"`},
		},
		{
			desc:    "reserved name 2",
			spec:    TransportSpec{Name: "Oneway"},
			wantErr: []string{`transport name cannot be "Oneway"`},
		},
		{
			desc:    "reserved name 3",
			spec:    TransportSpec{Name: "Stream"},
			wantErr: []string{`transport name cannot be "Stream"`},
		},
		{
			desc:    "missing BuildTransport",
			spec:    TransportSpec{Name: "foo"},
			wantErr: []string{"BuildTransport is required"},
		},
		{
			desc: "great sadness",
			spec: TransportSpec{
				Name:                "foo",
				BuildTransport:      func(struct{}, *Kit) (transport.Inbound, error) { panic("kthxbye") },
				BuildInbound:        func(transport.Transport) (transport.UnaryOutbound, error) { panic("kthxbye") },
				BuildUnaryOutbound:  func(struct{}, transport.Inbound, *Kit) (transport.UnaryOutbound, error) { panic("kthxbye") },
				BuildOnewayOutbound: func(struct{}) (transport.OnewayOutbound, error) { panic("kthxbye") },
				BuildStreamOutbound: func(struct{}) (transport.StreamOutbound, error) { panic("kthxbye") },
			},
			wantErr: []string{
				"invalid BuildTransport func(struct {}, *yarpcconfig.Kit) (transport.Inbound, error): " +
					"must return a transport.Transport as its first result, found transport.Inbound",
				"invalid BuildInbound: must accept exactly three arguments, found 1",
				"invalid BuildUnaryOutbound: must accept a transport.Transport as its second argument, found transport.Inbound",
				"invalid BuildOnewayOutbound: must accept exactly three arguments, found 1",
				"invalid BuildStreamOutbound: must accept exactly three arguments, found 1",
			},
		},
		{
			desc: "inbound only",
			spec: TransportSpec{
				Name:           "what-good-is-a-phone-call-when-you-are-unable-to-speak",
				BuildTransport: func(struct{}, *Kit) (transport.Transport, error) { panic("kthxbye") },
				BuildInbound:   func(*phoneCall, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			},
			transportInput: _typeOfEmptyStruct,
			inboundInput:   reflect.TypeOf(&phoneCall{}),
		},
		{
			desc: "unary outbound only",
			spec: TransportSpec{
				Name:               "tyrion",
				BuildTransport:     func(**struct{}, *Kit) (transport.Transport, error) { panic("kthxbye") },
				BuildUnaryOutbound: func(debt, transport.Transport, *Kit) (transport.UnaryOutbound, error) { panic("kthxbye") },
			},
			transportInput:     reflect.PtrTo(reflect.TypeOf(&struct{}{})),
			supportsUnary:      true,
			unaryOutboundInput: reflect.TypeOf(debt{}),
		},
		{
			desc: "oneway outbound only",
			spec: TransportSpec{
				Name:                "arise-riders-of-theoden",
				BuildTransport:      func(*cavalry, *Kit) (transport.Transport, error) { panic("kthxbye") },
				BuildOnewayOutbound: func(struct{}, transport.Transport, *Kit) (transport.OnewayOutbound, error) { panic("kthxbye") },
			},
			transportInput:      reflect.TypeOf(&cavalry{}),
			supportsOneway:      true,
			onewayOutboundInput: _typeOfEmptyStruct,
		},
		{
			desc: "stream outbound only",
			spec: TransportSpec{
				Name:                "arise-riders-of-theoden",
				BuildTransport:      func(*cavalry, *Kit) (transport.Transport, error) { panic("kthxbye") },
				BuildStreamOutbound: func(struct{}, transport.Transport, *Kit) (transport.StreamOutbound, error) { panic("kthxbye") },
			},
			transportInput:      reflect.TypeOf(&cavalry{}),
			supportsStream:      true,
			streamOutboundInput: _typeOfEmptyStruct,
		},
		{
			desc: "bad peer chooser preset",
			spec: TransportSpec{
				Name:               "foo",
				BuildTransport:     func(struct{}, *Kit) (transport.Transport, error) { panic("kthxbye") },
				BuildUnaryOutbound: func(struct{}, transport.Transport, *Kit) (transport.UnaryOutbound, error) { panic("kthxbye") },
				PeerChooserPresets: []PeerChooserPreset{
					{
						Name:             "fake",
						BuildPeerChooser: func(transport.Transport, *Kit) (peer.Chooser, error) { panic("kthxbye") },
					},
				},
			},
			wantErr: []string{
				`failed to compile preset for transport "foo":`,
				"invalid BuildPeerChooser func(transport.Transport, *yarpcconfig.Kit) (peer.Chooser, error):",
				"must accept a peer.Transport as its first argument, found transport.Transport",
			},
		},
		{
			desc: "peer chooser preset collision",
			spec: TransportSpec{
				Name:               "foo",
				BuildTransport:     func(struct{}, *Kit) (transport.Transport, error) { panic("kthxbye") },
				BuildUnaryOutbound: func(struct{}, transport.Transport, *Kit) (transport.UnaryOutbound, error) { panic("kthxbye") },
				PeerChooserPresets: []PeerChooserPreset{
					{
						Name:             "fake",
						BuildPeerChooser: func(peer.Transport, *Kit) (peer.Chooser, error) { panic("kthxbye") },
					},
					{
						Name:             "fake",
						BuildPeerChooser: func(peer.Transport, *Kit) (peer.Chooser, error) { panic("kthxbye") },
					},
				},
			},
			wantErr: []string{
				`found multiple peer lists with the name "fake" under transport "foo"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ts, err := compileTransportSpec(&tt.spec)

			if len(tt.wantErr) > 0 {
				if assert.Error(t, err, "expected failure") {
					for _, msg := range tt.wantErr {
						assert.Contains(t, err.Error(), msg)
					}
					for _, msg := range tt.wantErr {
						assert.Contains(t, fmt.Sprintf("%+v", err), msg)
					}
				}
				return
			}

			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tt.transportInput, ts.Transport.inputType)
			assert.Equal(t, tt.supportsUnary, ts.SupportsUnaryOutbound())
			assert.Equal(t, tt.supportsOneway, ts.SupportsOnewayOutbound())
			assert.Equal(t, tt.supportsStream, ts.SupportsStreamOutbound())

			if ts.Inbound != nil {
				assert.Equal(t, tt.inboundInput, ts.Inbound.inputType)
			}
			if ts.UnaryOutbound != nil {
				assert.Equal(t, tt.unaryOutboundInput, ts.UnaryOutbound.inputType)
			}
			if ts.OnewayOutbound != nil {
				assert.Equal(t, tt.onewayOutboundInput, ts.OnewayOutbound.inputType)
			}
			if ts.StreamOutbound != nil {
				assert.Equal(t, tt.streamOutboundInput, ts.StreamOutbound.inputType)
			}
		})
	}
}

func TestConfigSpecDecode(t *testing.T) {
	type item struct{ Key, Value string }

	someItem := item{"key", "value"}
	ptrToSomeItem := &someItem

	tests := []struct {
		desc string

		// Build funcction to compile
		build interface{}

		// Compile function to use (compileTransportConfig,
		// compileInboundConfig, etc.)
		compiler func(interface{}) (*configSpec, error)

		// Attributes to decode
		attrs config.AttributeMap

		// Whether we want a specific value decoded or an error message
		want    interface{}
		wantErr []string
	}{
		{
			desc:     "decode failure",
			build:    func(struct{}, *Kit) (transport.Transport, error) { panic("kthxbye") },
			compiler: compileTransportConfig,
			attrs:    config.AttributeMap{"unexpected": 42},
			wantErr: []string{
				"failed to decode struct {}",
				"has invalid keys: unexpected",
			},
		},
		{
			desc:     "decode struct{}",
			build:    func(struct{}, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			compiler: compileInboundConfig,
			attrs:    config.AttributeMap{},
			want:     struct{}{},
		},
		{
			desc:     "decode item",
			build:    func(item, transport.Transport, *Kit) (transport.UnaryOutbound, error) { panic("kthxbye") },
			compiler: compileUnaryOutboundConfig,
			attrs:    config.AttributeMap{"key": "key", "value": "value"},
			want:     someItem,
		},
		{
			desc:     "decode *item",
			build:    func(*item, transport.Transport, *Kit) (transport.UnaryOutbound, error) { panic("kthxbye") },
			compiler: compileUnaryOutboundConfig,
			attrs:    config.AttributeMap{"key": "key", "value": "value"},
			want:     ptrToSomeItem,
		},
		{
			desc:     "decode **item",
			build:    func(**item, transport.Transport, *Kit) (transport.UnaryOutbound, error) { panic("kthxbye") },
			compiler: compileUnaryOutboundConfig,
			attrs:    config.AttributeMap{"key": "key", "value": "value"},
			want:     &ptrToSomeItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			spec, err := tt.compiler(tt.build)
			if !assert.NoError(t, err, "failed to compile config") {
				return
			}

			got, err := spec.Decode(tt.attrs)
			if len(tt.wantErr) == 0 {
				if assert.NoError(t, err) {
					assert.Equal(t, tt.want, got.inputData.Interface())
				}
				return
			}

			if assert.Error(t, err, "expected failure") {
				for _, msg := range tt.wantErr {
					assert.Contains(t, err.Error(), msg)
				}
			}
		})
	}
}

// mockValueBuilder is a simple callable that records and verifies its calls using
// a gomock controller.
//
// mockValueBuilder.Build is a valid factory function for buildable.
type mockValueBuilder struct{ ctrl *gomock.Controller }

func newMockValueBuilder(ctrl *gomock.Controller) *mockValueBuilder {
	return &mockValueBuilder{ctrl: ctrl}
}

func (m *mockValueBuilder) ExpectBuild(args ...interface{}) *gomock.Call {
	return m.ctrl.RecordCall(m, "Build", args...)
}

func (m *mockValueBuilder) Build(args ...interface{}) (interface{}, error) {
	ret := m.ctrl.Call(m, "Build", args...)
	err, _ := ret[1].(error)
	return ret[0], err
}

func TestBuildableBuild(t *testing.T) {
	type item struct{ Key, Value string }

	tests := []struct {
		desc string

		// Configuration data and arguments for the build function
		data interface{}
		args []interface{}

		// Expect a Build(..) call with the given arguments
		wantArgs []interface{}
		err      error
	}{
		{
			desc:     "success, no args",
			data:     struct{}{},
			wantArgs: []interface{}{struct{}{}},
		},
		{
			desc:     "success with args",
			data:     1,
			args:     []interface{}{2, 3},
			wantArgs: []interface{}{1, 2, 3},
		},
		{
			desc: "success with Value args",
			data: &item{Key: "key", Value: "value"},
			args: []interface{}{
				"foo",
				reflect.ValueOf("bar"),
				"baz",
			},
			wantArgs: []interface{}{
				&item{Key: "key", Value: "value"},
				"foo",
				"bar",
				"baz",
			},
		},
		{
			desc:     "nil everything",
			data:     (*item)(nil),
			wantArgs: []interface{}{nil},
		},
		{
			desc:     "error no args",
			data:     42,
			wantArgs: []interface{}{42},
			err:      errors.New("great sadness"),
		},
		{
			desc: "error with args",
			data: item{},
			args: []interface{}{
				(*string)(nil),
				reflect.Zero(_typeOfTransport),
			},
			wantArgs: []interface{}{item{}, nil, nil},
			err:      errors.New("transport is required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			builder := newMockValueBuilder(mockCtrl)
			builder.ExpectBuild(tt.wantArgs...).Return("some result", tt.err)

			cv := &buildable{
				inputData: reflect.ValueOf(tt.data),
				factory:   reflect.ValueOf(builder.Build),
			}

			result, err := cv.Build(tt.args...)
			assert.Equal(t, tt.err, err)
			assert.Equal(t, "some result", result)
		})
	}
}

func TestCompileTransportConfig(t *testing.T) {
	tests := []struct {
		desc  string
		build interface{}

		wantInputType reflect.Type
		wantErr       string
	}{
		{
			desc:    "not a function",
			build:   42,
			wantErr: "must be a function",
		},
		{
			desc:    "wrong number of arguments",
			build:   func(struct{}, struct{}, struct{}) (transport.Transport, error) { panic("kthxbye") },
			wantErr: "must accept exactly two arguments, found 3",
		},
		{
			desc:    "incorrect input type",
			build:   func(int, *Kit) (transport.Transport, error) { panic("kthxbye") },
			wantErr: "must accept a struct or struct pointer as its first argument, found int",
		},
		{
			desc:    "wrong number of results",
			build:   func(struct{}, *Kit) transport.Transport { panic("kthxbye") },
			wantErr: "must return exactly two results, found 1",
		},
		{
			desc:    "wrong output type",
			build:   func(struct{}, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "must return a transport.Transport as its first result, found transport.Inbound",
		},
		{
			desc:    "incorrect second result",
			build:   func(struct{}, *Kit) (transport.Transport, string) { panic("kthxbye") },
			wantErr: "must return an error as its second result, found string",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}, *Kit) (transport.Transport, error) { panic("kthxbye") },
			wantInputType: _typeOfEmptyStruct,
		},
		{
			desc:          "valid: *struct{}",
			build:         func(*struct{}, *Kit) (transport.Transport, error) { panic("kthxbye") },
			wantInputType: reflect.PtrTo(_typeOfEmptyStruct),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cs, err := compileTransportConfig(tt.build)

			if tt.wantErr == "" {
				assert.Equal(t, tt.wantInputType, cs.inputType, "input type mismatch")
				assert.NoError(t, err, "expected success")
				return
			}

			if assert.Error(t, err, "expected failure") {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCompileInboundConfig(t *testing.T) {
	tests := []struct {
		desc          string
		build         interface{}
		wantInputType reflect.Type
		wantErr       string
	}{
		{
			desc:    "reserved field: Type",
			build:   func(struct{ Type string }, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "inbound configurations must not have a Type field",
		},
		{
			desc: "reserved field: Disabled",
			build: func(struct{ Disabled string }, transport.Transport, *Kit) (transport.Inbound, error) {
				panic("kthxbye")
			},
			wantErr: "inbound configurations must not have a Disabled field",
		},
		{
			desc:    "incorrect return type",
			build:   func(struct{}, transport.Transport, *Kit) (transport.Outbound, error) { panic("kthxbye") },
			wantErr: "invalid BuildInbound: must return a transport.Inbound as its first result, found transport.Outbound",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			wantInputType: _typeOfEmptyStruct,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cs, err := compileInboundConfig(tt.build)

			if tt.wantErr == "" {
				assert.Equal(t, tt.wantInputType, cs.inputType, "input type mismatch")
				assert.NoError(t, err, "expected success")
				return
			}

			if assert.Error(t, err, "expected failure") {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCompileUnaryOutboundConfig(t *testing.T) {
	tests := []struct {
		desc          string
		build         interface{}
		wantInputType reflect.Type
		wantErr       string
	}{
		{
			desc:    "incorrect return type",
			build:   func(struct{}, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "invalid BuildUnaryOutbound: must return a transport.UnaryOutbound as its first result, found transport.Inbound",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}, transport.Transport, *Kit) (transport.UnaryOutbound, error) { panic("kthxbye") },
			wantInputType: _typeOfEmptyStruct,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cs, err := compileUnaryOutboundConfig(tt.build)

			if tt.wantErr == "" {
				assert.Equal(t, tt.wantInputType, cs.inputType, "input type mismatch")
				assert.NoError(t, err, "expected success")
				return
			}

			if assert.Error(t, err, "expected failure") {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCompileOnewayOutboundConfig(t *testing.T) {
	tests := []struct {
		desc          string
		build         interface{}
		wantInputType reflect.Type
		wantErr       string
	}{
		{
			desc:    "incorrect return type",
			build:   func(struct{}, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "invalid BuildOnewayOutbound: must return a transport.OnewayOutbound as its first result, found transport.Inbound",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}, transport.Transport, *Kit) (transport.OnewayOutbound, error) { panic("kthxbye") },
			wantInputType: _typeOfEmptyStruct,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cs, err := compileOnewayOutboundConfig(tt.build)

			if tt.wantErr == "" {
				assert.Equal(t, tt.wantInputType, cs.inputType, "input type mismatch")
				assert.NoError(t, err, "expected success")
				return
			}

			if assert.Error(t, err, "expected failure") {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCompilePeerChooserSpec(t *testing.T) {
	tests := []struct {
		desc     string
		spec     PeerChooserSpec
		wantName string
		wantErr  string
	}{
		{
			desc:    "missing name",
			wantErr: "field Name is required",
		},
		{
			desc: "missing BuildPeerChooser",
			spec: PeerChooserSpec{
				Name: "random",
			},
			wantErr: "field BuildPeerChooser is required",
		},
		{
			desc: "not a function",
			spec: PeerChooserSpec{
				Name:             "much sadness",
				BuildPeerChooser: 10,
			},
			wantErr: "invalid BuildPeerChooser int: must be a function",
		},
		{
			desc: "too many arguments",
			spec: PeerChooserSpec{
				Name:             "much sadness",
				BuildPeerChooser: func(a, b, c, d int) {},
			},
			wantErr: "invalid BuildPeerChooser func(int, int, int, int): must accept exactly three arguments, found 4",
		},
		{
			desc: "wrong kind of first argument",
			spec: PeerChooserSpec{
				Name:             "much sadness",
				BuildPeerChooser: func(a, b, c int) {},
			},
			wantErr: "invalid BuildPeerChooser func(int, int, int): must accept a struct or struct pointer as its first argument, found int",
		},
		{
			desc: "wrong kind of second argument",
			spec: PeerChooserSpec{
				Name:             "much sadness",
				BuildPeerChooser: func(c struct{}, t int, k *Kit) {},
			},
			wantErr: "invalid BuildPeerChooser func(struct {}, int, *yarpcconfig.Kit): must accept a peer.Transport as its second argument, found int",
		},
		{
			desc: "wrong kind of third argument",
			spec: PeerChooserSpec{
				Name:             "much sadness",
				BuildPeerChooser: func(c struct{}, t peer.Transport, k int) {},
			},
			wantErr: "invalid BuildPeerChooser func(struct {}, peer.Transport, int): must accept a *yarpcconfig.Kit as its third argument, found int",
		},
		{
			desc: "wrong number of returns",
			spec: PeerChooserSpec{
				Name:             "much sadness",
				BuildPeerChooser: func(c struct{}, t peer.Transport, k *Kit) {},
			},
			wantErr: "invalid BuildPeerChooser func(struct {}, peer.Transport, *yarpcconfig.Kit): must return exactly two results, found 0",
		},
		{
			desc: "wrong type of first return",
			spec: PeerChooserSpec{
				Name: "much sadness",
				BuildPeerChooser: func(c struct{}, t peer.Transport, b *Kit) (int, error) {
					return 0, nil
				},
			},
			wantErr: "invalid BuildPeerChooser func(struct {}, peer.Transport, *yarpcconfig.Kit) (int, error): must return a peer.Chooser as its first result, found int",
		},
		{
			desc: "wrong type of second return",
			spec: PeerChooserSpec{
				Name: "much sadness",
				BuildPeerChooser: func(c struct{}, t peer.Transport, k *Kit) (peer.Chooser, int) {
					return nil, 0
				},
			},
			wantErr: "invalid BuildPeerChooser func(struct {}, peer.Transport, *yarpcconfig.Kit) (peer.Chooser, int): must return an error as its second result, found int",
		},
		{
			desc: "such gladness",
			spec: PeerChooserSpec{
				Name: "such gladness",
				BuildPeerChooser: func(c struct{}, t peer.Transport, k *Kit) (peer.Chooser, error) {
					return nil, nil
				},
			},
			wantName: "such gladness",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s, err := compilePeerChooserSpec(&tt.spec)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error(), "expected error")
			} else {
				assert.Equal(t, tt.wantName, s.Name, "expected name")
			}
		})
	}
}

func TestCompileStreamOutboundConfig(t *testing.T) {
	tests := []struct {
		desc          string
		build         interface{}
		wantInputType reflect.Type
		wantErr       string
	}{
		{
			desc:    "incorrect return type",
			build:   func(struct{}, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "invalid BuildStreamOutbound: must return a transport.StreamOutbound as its first result, found transport.Inbound",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}, transport.Transport, *Kit) (transport.StreamOutbound, error) { panic("kthxbye") },
			wantInputType: _typeOfEmptyStruct,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cs, err := compileStreamOutboundConfig(tt.build)

			if tt.wantErr == "" {
				assert.Equal(t, tt.wantInputType, cs.inputType, "input type mismatch")
				assert.NoError(t, err, "expected success")
				return
			}

			if assert.Error(t, err, "expected failure") {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCompilePeerListSpec(t *testing.T) {
	tests := []struct {
		desc     string
		spec     PeerListSpec
		wantName string
		wantErr  string
	}{
		{
			desc:    "missing name",
			wantErr: "field Name is required",
		},
		{
			desc: "missing BuildPeerList",
			spec: PeerListSpec{
				Name: "random",
			},
			wantErr: "field BuildPeerList is required",
		},
		{
			desc: "not a function",
			spec: PeerListSpec{
				Name:          "much sadness",
				BuildPeerList: 10,
			},
			wantErr: "invalid BuildPeerList int: must be a function",
		},
		{
			desc: "too many arguments",
			spec: PeerListSpec{
				Name:          "much sadness",
				BuildPeerList: func(a, b, c, d int) {},
			},
			wantErr: "invalid BuildPeerList func(int, int, int, int): must accept exactly three arguments, found 4",
		},
		{
			desc: "wrong kind of first argument",
			spec: PeerListSpec{
				Name:          "much sadness",
				BuildPeerList: func(a, b, c int) {},
			},
			wantErr: "invalid BuildPeerList func(int, int, int): must accept a struct or struct pointer as its first argument, found int",
		},
		{
			desc: "wrong kind of second argument",
			spec: PeerListSpec{
				Name:          "much sadness",
				BuildPeerList: func(c struct{}, t int, k *Kit) {},
			},
			wantErr: "invalid BuildPeerList func(struct {}, int, *yarpcconfig.Kit): must accept a peer.Transport as its second argument, found int",
		},
		{
			desc: "wrong kind of third argument",
			spec: PeerListSpec{
				Name:          "much sadness",
				BuildPeerList: func(c struct{}, t peer.Transport, k int) {},
			},
			wantErr: "invalid BuildPeerList func(struct {}, peer.Transport, int): must accept a *yarpcconfig.Kit as its third argument, found int",
		},
		{
			desc: "wrong number of returns",
			spec: PeerListSpec{
				Name:          "much sadness",
				BuildPeerList: func(c struct{}, t peer.Transport, k *Kit) {},
			},
			wantErr: "invalid BuildPeerList func(struct {}, peer.Transport, *yarpcconfig.Kit): must return exactly two results, found 0",
		},
		{
			desc: "wrong type of first return",
			spec: PeerListSpec{
				Name: "much sadness",
				BuildPeerList: func(c struct{}, t peer.Transport, b *Kit) (int, error) {
					return 0, nil
				},
			},
			wantErr: "invalid BuildPeerList func(struct {}, peer.Transport, *yarpcconfig.Kit) (int, error): must return a peer.ChooserList as its first result, found int",
		},
		{
			desc: "wrong type of second return",
			spec: PeerListSpec{
				Name: "much sadness",
				BuildPeerList: func(c struct{}, t peer.Transport, k *Kit) (peer.ChooserList, int) {
					return nil, 0
				},
			},
			wantErr: "invalid BuildPeerList func(struct {}, peer.Transport, *yarpcconfig.Kit) (peer.ChooserList, int): must return an error as its second result, found int",
		},
		{
			desc: "such gladness",
			spec: PeerListSpec{
				Name: "such gladness",
				BuildPeerList: func(c struct{}, t peer.Transport, k *Kit) (peer.ChooserList, error) {
					return nil, nil
				},
			},
			wantName: "such gladness",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s, err := compilePeerListSpec(&tt.spec)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error(), "expected error")
			} else {
				assert.Equal(t, tt.wantName, s.Name, "expected name")
			}
		})
	}
}

func TestCompilePeerListUpdaterSpec(t *testing.T) {
	tests := []struct {
		desc     string
		spec     PeerListUpdaterSpec
		wantName string
		wantErr  string
	}{
		{
			desc:    "missing name",
			wantErr: "field Name is required",
		},
		{
			desc: "missing BuildPeerListUpdater",
			spec: PeerListUpdaterSpec{
				Name: "random",
			},
			wantErr: "field BuildPeerListUpdater is required",
		},
		{
			desc: "not a function",
			spec: PeerListUpdaterSpec{
				Name:                 "much sadness",
				BuildPeerListUpdater: 10,
			},
			wantErr: "invalid BuildPeerListUpdater int: must be a function",
		},
		{
			desc: "too many arguments",
			spec: PeerListUpdaterSpec{
				Name:                 "much sadness",
				BuildPeerListUpdater: func(a, b, c int) {},
			},
			wantErr: "invalid BuildPeerListUpdater func(int, int, int): must accept exactly two arguments, found 3",
		},
		{
			desc: "wrong kind of first argument",
			spec: PeerListUpdaterSpec{
				Name:                 "much sadness",
				BuildPeerListUpdater: func(a, b int) {},
			},
			wantErr: "invalid BuildPeerListUpdater func(int, int): must accept a struct or struct pointer as its first argument, found int",
		},
		{
			desc: "wrong kind of second argument",
			spec: PeerListUpdaterSpec{
				Name:                 "much sadness",
				BuildPeerListUpdater: func(a struct{}, b int) {},
			},
			wantErr: "invalid BuildPeerListUpdater func(struct {}, int): must accept a *yarpcconfig.Kit as its second argument, found int",
		},
		{
			desc: "wrong number of returns",
			spec: PeerListUpdaterSpec{
				Name:                 "much sadness",
				BuildPeerListUpdater: func(a struct{}, b *Kit) {},
			},
			wantErr: "invalid BuildPeerListUpdater func(struct {}, *yarpcconfig.Kit): must return exactly two results, found 0",
		},
		{
			desc: "wrong type of first return",
			spec: PeerListUpdaterSpec{
				Name: "much sadness",
				BuildPeerListUpdater: func(a struct{}, b *Kit) (int, error) {
					return 0, nil
				},
			},
			wantErr: "invalid BuildPeerListUpdater func(struct {}, *yarpcconfig.Kit) (int, error): must return a peer.Binder as its first result, found int",
		},
		{
			desc: "wrong type of second return",
			spec: PeerListUpdaterSpec{
				Name: "much sadness",
				BuildPeerListUpdater: func(a struct{}, b *Kit) (peer.Binder, int) {
					return nil, 0
				},
			},
			wantErr: "invalid BuildPeerListUpdater func(struct {}, *yarpcconfig.Kit) (peer.Binder, int): must return an error as its second result, found int",
		},
		{
			desc: "such gladness",
			spec: PeerListUpdaterSpec{
				Name: "such gladness",
				BuildPeerListUpdater: func(a struct{}, b *Kit) (peer.Binder, error) {
					return nil, nil
				},
			},
			wantName: "such gladness",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s, err := compilePeerListUpdaterSpec(&tt.spec)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error(), "expected error")
			} else {
				assert.Equal(t, tt.wantName, s.Name, "expected name")
			}
		})
	}
}

func TestCompilePeerChooserPreset(t *testing.T) {
	tests := []struct {
		desc     string
		spec     PeerChooserPreset
		wantName string
		wantErr  string
	}{
		{
			desc:    "missing name",
			wantErr: "field Name is required",
		},
		{
			desc: "missing BuildPeerChooser",
			spec: PeerChooserPreset{
				Name: "random",
			},
			wantErr: "field BuildPeerChooser is required",
		},
		{
			desc: "not a function",
			spec: PeerChooserPreset{
				Name:             "much sadness",
				BuildPeerChooser: 10,
			},
			wantErr: "invalid BuildPeerChooser int: must be a function",
		},
		{
			desc: "too many arguments",
			spec: PeerChooserPreset{
				Name:             "much sadness",
				BuildPeerChooser: func(a, b, c, d int) {},
			},
			wantErr: "invalid BuildPeerChooser func(int, int, int, int): must accept exactly two arguments, found 4",
		},
		{
			desc: "wrong kind of first argument",
			spec: PeerChooserPreset{
				Name:             "much sadness",
				BuildPeerChooser: func(a, b int) {},
			},
			wantErr: "invalid BuildPeerChooser func(int, int): must accept a peer.Transport as its first argument, found int",
		},
		{
			desc: "wrong kind of second",
			spec: PeerChooserPreset{
				Name:             "much sadness",
				BuildPeerChooser: func(peer.Transport, int) {},
			},
			wantErr: "invalid BuildPeerChooser func(peer.Transport, int): must accept a *yarpcconfig.Kit as its second argument, found int",
		},
		{
			desc: "wrong number of returns",
			spec: PeerChooserPreset{
				Name:             "much sadness",
				BuildPeerChooser: func(t peer.Transport, k *Kit) {},
			},
			wantErr: "invalid BuildPeerChooser func(peer.Transport, *yarpcconfig.Kit): must return exactly two results, found 0",
		},
		{
			desc: "wrong type of first return",
			spec: PeerChooserPreset{
				Name: "much sadness",
				BuildPeerChooser: func(t peer.Transport, b *Kit) (int, error) {
					return 0, nil
				},
			},
			wantErr: "invalid BuildPeerChooser func(peer.Transport, *yarpcconfig.Kit) (int, error): must return a peer.Chooser as its first result, found int",
		},
		{
			desc: "wrong type of second return",
			spec: PeerChooserPreset{
				Name: "much sadness",
				BuildPeerChooser: func(t peer.Transport, k *Kit) (peer.Chooser, int) {
					return nil, 0
				},
			},
			wantErr: "invalid BuildPeerChooser func(peer.Transport, *yarpcconfig.Kit) (peer.Chooser, int): must return an error as its second result, found int",
		},
		{
			desc: "such gladness",
			spec: PeerChooserPreset{
				Name: "such gladness",
				BuildPeerChooser: func(t peer.Transport, k *Kit) (peer.Chooser, error) {
					return nil, nil
				},
			},
			wantName: "such gladness",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s, err := compilePeerChooserPreset(tt.spec)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error(), "expected error")
			} else {
				assert.Equal(t, tt.wantName, s.name, "expected name")
			}
		})
	}
}

func TestValidateConfigFunc(t *testing.T) {
	tests := []struct {
		desc string

		// Build function. We'll use its type for the test.
		build interface{}

		// Type of output expected from the function
		outputType reflect.Type

		// If non-empty, we expect an error
		wantErr string
	}{
		{
			desc:       "not a function",
			build:      42,
			outputType: _typeOfEmptyStruct,
			wantErr:    "must be a function",
		},
		{
			desc:       "wrong number of arguments",
			build:      func(struct{}) (transport.Inbound, error) { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must accept exactly three arguments, found 1",
		},
		{
			desc:       "incorrect input type",
			build:      func(int, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must accept a struct or struct pointer as its first argument, found int",
		},
		{
			desc:       "incorrect second argument",
			build:      func(struct{}, int, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must accept a transport.Transport as its second argument, found int",
		},
		{
			desc:       "wrong number of results",
			build:      func(struct{}, transport.Transport, *Kit) transport.Inbound { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must return exactly two results, found 1",
		},
		{
			desc:       "wrong output type",
			build:      func(struct{}, transport.Transport, *Kit) (transport.Inbound, error) { panic("kthxbye") },
			outputType: _typeOfUnaryOutbound,
			wantErr:    "must return a transport.UnaryOutbound as its first result, found transport.Inbound",
		},
		{
			desc:       "incorrect second result",
			build:      func(struct{}, transport.Transport, *Kit) (transport.Inbound, string) { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must return an error as its second result, found string",
		},
		{
			desc:       "valid",
			build:      func(struct{}, transport.Transport, *Kit) (struct{}, error) { panic("kthxbye") },
			outputType: _typeOfEmptyStruct,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			funcType := reflect.TypeOf(tt.build)
			err := validateConfigFunc(funcType, tt.outputType)

			if tt.wantErr == "" {
				assert.NoError(t, err, "expected success")
				return
			}

			if assert.Error(t, err, "expected failure") {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestFieldNames(t *testing.T) {
	tests := []struct {
		give reflect.Type
		want []string
	}{
		{give: _typeOfEmptyStruct},
		{give: _typeOfError},
		{
			give: reflect.TypeOf(struct {
				Name   string
				Value  string
				hidden int64
			}{}),
			want: []string{"Name", "Value"},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.give), func(t *testing.T) {
			want := make(map[string]struct{})
			for _, f := range tt.want {
				want[f] = struct{}{}
			}

			if len(want) == 0 {
				// play nicely with nil/empty
				assert.Empty(t, fieldNames(tt.give))
			} else {
				assert.Equal(t, want, fieldNames(tt.give))
			}
		})
	}
}

func TestIsDecodable(t *testing.T) {
	tests := []struct {
		give reflect.Type
		want bool
	}{
		{give: _typeOfError, want: false},
		{give: reflect.PtrTo(_typeOfError), want: false},
		{give: _typeOfEmptyStruct, want: true},
		{give: reflect.PtrTo(_typeOfEmptyStruct), want: true},
		{give: reflect.PtrTo(reflect.PtrTo(_typeOfEmptyStruct)), want: true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.give), func(t *testing.T) {
			assert.Equal(t, tt.want, isDecodable(tt.give))
		})
	}
}
