package config

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"go.uber.org/yarpc/api/transport"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

		transportInput      reflect.Type
		inboundInput        reflect.Type
		unaryOutboundInput  reflect.Type
		onewayOutboundInput reflect.Type

		wantErr []string
	}{
		{
			desc:    "missing name",
			wantErr: []string{"Name is required"},
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
				BuildTransport:      func(struct{}) (transport.Inbound, error) { panic("kthxbye") },
				BuildInbound:        func(transport.Transport) (transport.UnaryOutbound, error) { panic("kthxbye") },
				BuildUnaryOutbound:  func(struct{}, transport.Inbound) (transport.UnaryOutbound, error) { panic("kthxbye") },
				BuildOnewayOutbound: func(struct{}) (transport.OnewayOutbound, error) { panic("kthxbye") },
			},
			wantErr: []string{
				"the following errors occurred:",
				"invalid BuildTransport func(struct {}) (transport.Inbound, error): " +
					"must return a transport.Transport as its first result, found transport.Inbound",
				"invalid BuildInbound: must accept exactly two arguments, found 1",
				"invalid BuildUnaryOutbound: must accept a transport.Transport as its second argument, found transport.Inbound",
				"invalid BuildOnewayOutbound: must accept exactly two arguments, found 1",
			},
		},
		{
			desc: "inbound only",
			spec: TransportSpec{
				Name:           "what-good-is-a-phone-call-when-you-are-unable-to-speak",
				BuildTransport: func(struct{}) (transport.Transport, error) { panic("kthxbye") },
				BuildInbound:   func(*phoneCall, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
			},
			transportInput: _typeOfEmptyStruct,
			inboundInput:   reflect.TypeOf(&phoneCall{}),
		},
		{
			desc: "unary outbound only",
			spec: TransportSpec{
				Name:               "tyrion",
				BuildTransport:     func(**struct{}) (transport.Transport, error) { panic("kthxbye") },
				BuildUnaryOutbound: func(debt, transport.Transport) (transport.UnaryOutbound, error) { panic("kthxbye") },
			},
			transportInput:     reflect.PtrTo(reflect.TypeOf(&struct{}{})),
			supportsUnary:      true,
			unaryOutboundInput: reflect.TypeOf(debt{}),
		},
		{
			desc: "oneway outbound only",
			spec: TransportSpec{
				Name:                "arise-riders-of-theoden",
				BuildTransport:      func(*cavalry) (transport.Transport, error) { panic("kthxbye") },
				BuildOnewayOutbound: func(struct{}, transport.Transport) (transport.OnewayOutbound, error) { panic("kthxbye") },
			},
			transportInput:      reflect.TypeOf(&cavalry{}),
			supportsOneway:      true,
			onewayOutboundInput: _typeOfEmptyStruct,
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
				}
				return
			}

			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tt.transportInput, ts.Transport.inputType)
			assert.Equal(t, tt.supportsUnary, ts.SupportsUnaryOutbound())
			assert.Equal(t, tt.supportsOneway, ts.SupportsOnewayOutbound())

			if ts.Inbound != nil {
				assert.Equal(t, tt.inboundInput, ts.Inbound.inputType)
			}
			if ts.UnaryOutbound != nil {
				assert.Equal(t, tt.unaryOutboundInput, ts.UnaryOutbound.inputType)
			}
			if ts.OnewayOutbound != nil {
				assert.Equal(t, tt.onewayOutboundInput, ts.OnewayOutbound.inputType)
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
		attrs attributeMap

		// Whether we want a specific value decoded or an error message
		want    interface{}
		wantErr []string
	}{
		{
			desc:     "decode failure",
			build:    func(struct{}) (transport.Transport, error) { panic("kthxbye") },
			compiler: compileTransportConfig,
			attrs:    attributeMap{"unexpected": 42},
			wantErr: []string{
				"failed to decode struct {}",
				"has invalid keys: unexpected",
			},
		},
		{
			desc:     "decode struct{}",
			build:    func(struct{}, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
			compiler: compileInboundConfig,
			attrs:    attributeMap{},
			want:     struct{}{},
		},
		{
			desc:     "decode item",
			build:    func(item, transport.Transport) (transport.UnaryOutbound, error) { panic("kthxbye") },
			compiler: compileUnaryOutboundConfig,
			attrs:    attributeMap{"key": "key", "value": "value"},
			want:     someItem,
		},
		{
			desc:     "decode *item",
			build:    func(*item, transport.Transport) (transport.UnaryOutbound, error) { panic("kthxbye") },
			compiler: compileUnaryOutboundConfig,
			attrs:    attributeMap{"key": "key", "value": "value"},
			want:     ptrToSomeItem,
		},
		{
			desc:     "decode **item",
			build:    func(**item, transport.Transport) (transport.UnaryOutbound, error) { panic("kthxbye") },
			compiler: compileUnaryOutboundConfig,
			attrs:    attributeMap{"key": "key", "value": "value"},
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
					assert.Equal(t, tt.want, got.data.Interface())
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

// mockBuilder is a simple callable that records and verifies its calls using
// a gomock controller.
//
// m.Build is a valid builder function for configuredValue for some
// mockBuilder m.
type mockBuilder struct{ ctrl *gomock.Controller }

func newMockBuilder(ctrl *gomock.Controller) *mockBuilder {
	return &mockBuilder{ctrl: ctrl}
}

func (m *mockBuilder) ExpectBuild(args ...interface{}) *gomock.Call {
	return m.ctrl.RecordCall(m, "Build", args...)
}

func (m *mockBuilder) Build(args ...interface{}) (interface{}, error) {
	ret := m.ctrl.Call(m, "Build", args...)
	err, _ := ret[1].(error)
	return ret[0], err
}

func TestConfiguredValueDecode(t *testing.T) {
	type item struct{ Key, Value string }

	tests := []struct {
		desc string

		// Configuration data and arguments for the build function
		data interface{}
		args []interface{}

		// Expect a Build(..) call with the given arguments
		wantArgs []interface{}

		// Result and error of calling the build function
		result interface{}
		err    error
	}{
		{
			desc:     "success, no args",
			data:     struct{}{},
			wantArgs: []interface{}{struct{}{}},
			result:   42,
		},
		{
			desc:     "success with args",
			data:     1,
			args:     []interface{}{2, 3},
			wantArgs: []interface{}{1, 2, 3},
			result:   4,
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
			result: `¯\_(ツ)_/¯`,
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

			builder := newMockBuilder(mockCtrl)
			builder.ExpectBuild(tt.wantArgs...).Return(tt.result, tt.err)

			cv := &configuredValue{
				data:    reflect.ValueOf(tt.data),
				builder: reflect.ValueOf(builder.Build),
			}

			result, err := cv.Build(tt.args...)
			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.result, result)
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
			build:   func(struct{}, struct{}) (transport.Transport, error) { panic("kthxbye") },
			wantErr: "must accept exactly one argument, found 2",
		},
		{
			desc:    "incorrect input type",
			build:   func(int) (transport.Transport, error) { panic("kthxbye") },
			wantErr: "must accept a struct or struct pointer as its first argument, found int",
		},
		{
			desc:    "wrong number of results",
			build:   func(struct{}) transport.Transport { panic("kthxbye") },
			wantErr: "must return exactly two results, found 1",
		},
		{
			desc:    "wrong output type",
			build:   func(struct{}) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "must return a transport.Transport as its first result, found transport.Inbound",
		},
		{
			desc:    "incorrect second result",
			build:   func(struct{}) (transport.Transport, string) { panic("kthxbye") },
			wantErr: "must return an error as its second result, found string",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}) (transport.Transport, error) { panic("kthxbye") },
			wantInputType: _typeOfEmptyStruct,
		},
		{
			desc:          "valid: *struct{}",
			build:         func(*struct{}) (transport.Transport, error) { panic("kthxbye") },
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
			build:   func(struct{ Type string }, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "inbound configurations must not have a Type field",
		},
		{
			desc:    "reserved field: Disabled",
			build:   func(struct{ Disabled string }, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "inbound configurations must not have a Disabled field",
		},
		{
			desc:    "incorrect return type",
			build:   func(struct{}, transport.Transport) (transport.Outbound, error) { panic("kthxbye") },
			wantErr: "invalid BuildInbound: must return a transport.Inbound as its first result, found transport.Outbound",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
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
			build:   func(struct{}, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "invalid BuildUnaryOutbound: must return a transport.UnaryOutbound as its first result, found transport.Inbound",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}, transport.Transport) (transport.UnaryOutbound, error) { panic("kthxbye") },
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
			build:   func(struct{}, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
			wantErr: "invalid BuildOnewayOutbound: must return a transport.OnewayOutbound as its first result, found transport.Inbound",
		},
		{
			desc:          "valid: struct{}",
			build:         func(struct{}, transport.Transport) (transport.OnewayOutbound, error) { panic("kthxbye") },
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
			wantErr:    "must accept exactly two arguments, found 1",
		},
		{
			desc:       "incorrect input type",
			build:      func(int, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must accept a struct or struct pointer as its first argument, found int",
		},
		{
			desc:       "incorrect second argument",
			build:      func(struct{}, int) (transport.Inbound, error) { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must accept a transport.Transport as its second argument, found int",
		},
		{
			desc:       "wrong number of results",
			build:      func(struct{}, transport.Transport) transport.Inbound { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must return exactly two results, found 1",
		},
		{
			desc:       "wrong output type",
			build:      func(struct{}, transport.Transport) (transport.Inbound, error) { panic("kthxbye") },
			outputType: _typeOfUnaryOutbound,
			wantErr:    "must return a transport.UnaryOutbound as its first result, found transport.Inbound",
		},
		{
			desc:       "incorrect second result",
			build:      func(struct{}, transport.Transport) (transport.Inbound, string) { panic("kthxbye") },
			outputType: _typeOfInbound,
			wantErr:    "must return an error as its second result, found string",
		},
		{
			desc:       "valid",
			build:      func(struct{}, transport.Transport) (struct{}, error) { panic("kthxbye") },
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
