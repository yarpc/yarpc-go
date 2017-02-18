package config

import (
	"errors"
	"fmt"

	"go.uber.org/yarpc/internal/decode"
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

	err = decode.Decode(dst, v)
	if err != nil {
		err = fmt.Errorf("failed to read attribute %q: %v", name, v)
	}
	return true, err
}

func (m attributeMap) Decode(dst interface{}) error {
	return decode.Decode(dst, m)
}

type yarpcConfig struct {
	Name       string                  `config:"name"`
	Inbounds   inbounds                `config:"inbounds"`
	Outbounds  clientConfigs           `config:"outbounds"`
	Transports map[string]attributeMap `config:"transports"`
}

type inbounds []inbound

func (is *inbounds) Decode(decode decode.Into) error {
	var items map[string]inbound
	if err := decode(&items); err != nil {
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

func (i *inbound) Decode(decode decode.Into) error {
	if err := decode(&i.Attributes); err != nil {
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

func (cc *clientConfigs) Decode(decode decode.Into) error {
	var items map[string]outbounds
	if err := decode(&items); err != nil {
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

func (o *outbounds) Decode(decode decode.Into) error {
	var attrs attributeMap
	if err := decode(&attrs); err != nil {
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
	Preset     string
	Attributes attributeMap
}

func (o *outbound) Decode(decode decode.Into) error {
	var cfg map[string]attributeMap
	if err := decode(&cfg); err != nil {
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
		var err error

		o.Type = k
		o.Attributes = attrs
		o.Preset, err = attrs.PopString("with")
		if err != nil {
			return fmt.Errorf(`failed to decode outbound attribute "with": %v`, err)
		}
	}

	return nil
}
