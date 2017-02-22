// Copyright (c) 2017 Uber Technologies, Inc.
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

package config

import (
	"errors"
	"fmt"
	"reflect"
)

// TransportSpec TODO
type TransportSpec struct {
	Name                string
	TransportConfigType TransportBuilder

	// Everything below is optional

	InboundConfigType        InboundBuilder
	UnaryOutboundConfigType  UnaryOutboundBuilder
	OnewayOutboundConfigType OnewayOutboundBuilder
	UnaryOutboundPresets     map[string]UnaryOutboundBuilder
	OnewayOutboundPresets    map[string]OnewayOutboundBuilder
}

func getStruct(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Struct {
		return t
	}

	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		return t.Elem()
	}

	return nil
}

// RegisterTransport TODO
func (l *Loader) RegisterTransport(t TransportSpec) error {
	spec := transportSpec{name: t.Name}

	// TODO include more information in error

	if t.TransportConfigType == nil {
		return errors.New("a transport configuration type is required")
	}

	spec.transportConfigType = getStruct(reflect.TypeOf(t.TransportConfigType))
	if spec.transportConfigType == nil {
		return errors.New("transport configurations can only be defined on structs")
	}

	if t.InboundConfigType != nil {
		spec.inboundConfigType = getStruct(reflect.TypeOf(t.InboundConfigType))
		if spec.inboundConfigType == nil {
			return errors.New("inbound configurations can only be defined on structs")
		}

		if _, ok := spec.inboundConfigType.FieldByName("Type"); ok {
			return fmt.Errorf("inbound configurations cannot have a Type field")
		}

		if _, ok := spec.inboundConfigType.FieldByName("Disabled"); ok {
			return fmt.Errorf("inbound configurations cannot have a Disabled field")
		}
	}

	if t.UnaryOutboundConfigType != nil {
		spec.unaryOutboundConfigType = getStruct(reflect.TypeOf(t.UnaryOutboundConfigType))
		if spec.unaryOutboundConfigType == nil {
			return fmt.Errorf("unary outbound configurations can only be defined on structs")
		}

		// We should be checking the config: tags too
		if _, ok := spec.unaryOutboundConfigType.FieldByName("With"); ok {
			return fmt.Errorf("outbound configurations cannot have a With field")
		}
	}

	if t.OnewayOutboundConfigType != nil {
		spec.onewayOutboundConfigType = getStruct(reflect.TypeOf(t.OnewayOutboundConfigType))
		if spec.onewayOutboundConfigType == nil {
			return fmt.Errorf("oneway outbound configurations can only be defined on structs")
		}

		if _, ok := spec.onewayOutboundConfigType.FieldByName("With"); ok {
			return fmt.Errorf("outbound configurations cannot have a With field")
		}
	}

	spec.unaryOutboundPresets = make(map[string]reflect.Type, len(t.UnaryOutboundPresets))
	for name, preset := range t.UnaryOutboundPresets {
		spec.unaryOutboundPresets[name] = getStruct(reflect.TypeOf(preset))
		if spec.unaryOutboundPresets[name] == nil {
			return fmt.Errorf("outbound presets can only be defined on structs")
		}
	}

	spec.onewayOutboundPresets = make(map[string]reflect.Type, len(t.OnewayOutboundPresets))
	for name, preset := range t.OnewayOutboundPresets {
		spec.onewayOutboundPresets[name] = getStruct(reflect.TypeOf(preset))
		if spec.onewayOutboundPresets[name] == nil {
			return fmt.Errorf("outbound presets can only be defined on structs")
		}
	}

	// TODO: Panic if a transport with the given name is already registered?
	l.knownTransports[t.Name] = spec
	return nil
}
