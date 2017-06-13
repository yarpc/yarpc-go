package config

import (
	"fmt"

	"github.com/uber-go/mapdecode"
)

// AttributeMap is a convenience type on top of a map
// that gives us a cleaner interface to validate config values.
type AttributeMap map[string]interface{}

// PopString will pop a value from the attribute map and return the string
// it points to, or an error if it couldn't pop the value and decode.
func (m AttributeMap) PopString(name string) (s string, err error) {
	_, err = m.Pop(name, &s)
	return
}

// PopBool will pop a value from the attribute map and return the bool
// it points to, or an error if it couldn't pop the value and decode.
func (m AttributeMap) PopBool(name string) (b bool, err error) {
	_, err = m.Pop(name, &b)
	return
}

// Pop removes the named key from the AttributeMap and decodes the value into
// the dst interface.
func (m AttributeMap) Pop(name string, dst interface{}) (ok bool, err error) {
	ok, err = m.Get(name, dst)
	if ok {
		delete(m, name)
	}
	return
}

// Get grabs a value from the attribute map and decodes it into the dst
// interface.
func (m AttributeMap) Get(name string, dst interface{}) (ok bool, err error) {
	v, ok := m[name]
	if !ok {
		return ok, nil
	}

	err = DecodeInto(dst, v)
	if err != nil {
		err = fmt.Errorf("failed to read attribute %q: %v", name, v)
	}
	return true, err
}

// Keys returns all the keys of the attribute map.
func (m AttributeMap) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// Decode attempts to decode the AttributeMap into the dst interface.
func (m AttributeMap) Decode(dst interface{}, opts ...mapdecode.Option) error {
	return DecodeInto(dst, m, opts...)
}
