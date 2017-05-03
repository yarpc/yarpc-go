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

	"github.com/uber-go/mapdecode"
)

type attributeMap map[string]interface{}

func (m attributeMap) PopString(name string) (s string, err error) {
	_, err = m.Pop(name, &s)
	return
}

func (m attributeMap) PopBool(name string) (b bool, err error) {
	_, err = m.Pop(name, &b)
	return
}

func (m attributeMap) Pop(name string, dst interface{}) (ok bool, err error) {
	ok, err = m.Get(name, dst)
	if ok {
		delete(m, name)
	}
	return
}

func (m attributeMap) Get(name string, dst interface{}) (ok bool, err error) {
	v, ok := m[name]
	if !ok {
		return ok, nil
	}

	err = decodeInto(dst, v)
	if err != nil {
		err = fmt.Errorf("failed to read attribute %q: %v", name, v)
	}
	return true, err
}

func (m attributeMap) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func (m attributeMap) Decode(dst interface{}, opts ...mapdecode.Option) error {
	return decodeInto(dst, m, opts...)
}

type yarpcConfig struct {
	Inbounds   inbounds                `config:"inbounds"`
	Outbounds  clientConfigs           `config:"outbounds"`
	Transports map[string]attributeMap `config:"transports"`
}

type inbounds []inbound

func (is *inbounds) Decode(into mapdecode.Into) error {
	var items map[string]inbound
	if err := into(&items); err != nil {
		return fmt.Errorf("failed to decode inbound items: %v", err)
	}

	for k, v := range items {
		if v.Type == "" {
			v.Type = k
		}
		*is = append(*is, v)
	}
	return nil
}

type inbound struct {
	Type       string
	Disabled   bool
	Attributes attributeMap
}

func (i *inbound) Decode(into mapdecode.Into) error {
	if err := into(&i.Attributes); err != nil {
		return fmt.Errorf("failed to decode inbound: %v", err)
	}

	var err error
	i.Type, err = i.Attributes.PopString("type")
	if err != nil {
		return fmt.Errorf(`failed to read attribute "type" of inbound: %v`, err)
	}
	i.Disabled, err = i.Attributes.PopBool("disabled")
	if err != nil {
		return fmt.Errorf(`failed to read attribute "disabled" of inbound: %v`, err)
	}

	return nil
}

type clientConfigs map[string]outbounds

func (cc *clientConfigs) Decode(into mapdecode.Into) error {
	var items map[string]outbounds
	if err := into(&items); err != nil {
		return fmt.Errorf("failed to decode outbound items: %v", err)
	}

	for k, v := range items {
		if v.Service == "" {
			v.Service = k
		}
		items[k] = v
	}
	*cc = items
	return nil
}

type outbounds struct {
	Service string

	// Either (Unary and/or Oneway) will be set or Implicit will be set. For
	// the latter case, we need to only use those configurations that that
	// transport supports.
	Unary    *outbound
	Oneway   *outbound
	Implicit *outbound
}

func (o *outbounds) Decode(into mapdecode.Into) error {
	var attrs attributeMap
	if err := into(&attrs); err != nil {
		return fmt.Errorf("failed to decode outbound configuration: %v", err)
	}

	var err error
	o.Service, err = attrs.PopString("service")
	if err != nil {
		return fmt.Errorf("failed to read service name for outbound: %v", err)
	}

	hasUnary, err := attrs.Pop("unary", &o.Unary)
	if err != nil {
		return fmt.Errorf("failed to unary outbound configuration: %v", err)
	}

	hasOneway, err := attrs.Pop("oneway", &o.Oneway)
	if err != nil {
		return fmt.Errorf("failed to oneway outbound configuration: %v", err)
	}

	if hasUnary || hasOneway {
		// No more attributes should be remaining
		var empty struct{}
		if err := attrs.Decode(&empty); err != nil {
			return fmt.Errorf(
				"too many attributes in explicit outbound configuration: %v", err)
		}
		return nil
	}

	if err := attrs.Decode(&o.Implicit); err != nil {
		return fmt.Errorf("failed to decode implicit outbound configuration: %v", err)
	}
	return nil
}

type outbound struct {
	Type       string
	Attributes attributeMap
}

func (o *outbound) Decode(into mapdecode.Into) error {
	var cfg map[string]attributeMap
	if err := into(&cfg); err != nil {
		return fmt.Errorf("failed to decode outbound: %v", err)
	}

	switch len(cfg) {
	case 0:
		return errors.New("failed to decode outbound: an outbound type is required")
	case 1:
		// Move along
	default:
		return errors.New("failed to decode outbound: " +
			"at most one outbound type may be specified")
	}

	for k, attrs := range cfg {
		o.Type = k
		o.Attributes = attrs
	}

	return nil
}
