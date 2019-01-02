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

package yarpcconfig

import (
	"errors"
	"fmt"

	"github.com/uber-go/mapdecode"
	"go.uber.org/yarpc/internal/config"
)

type yarpcConfig struct {
	Inbounds   inbounds                       `config:"inbounds"`
	Outbounds  clientConfigs                  `config:"outbounds"`
	Transports map[string]config.AttributeMap `config:"transports"`
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
	Attributes config.AttributeMap
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
	Stream   *outbound
	Implicit *outbound
}

func (o *outbounds) Decode(into mapdecode.Into) error {
	var attrs config.AttributeMap
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

	hasStream, err := attrs.Pop("stream", &o.Stream)
	if err != nil {
		return fmt.Errorf("failed to stream outbound configuration: %v", err)
	}

	if hasUnary || hasOneway || hasStream {
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
	Attributes config.AttributeMap
}

func (o *outbound) Decode(into mapdecode.Into) error {
	var cfg map[string]config.AttributeMap
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
