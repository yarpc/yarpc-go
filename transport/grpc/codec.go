package grpc

import "fmt"

// passThroughCodec passes bytes to/from the wire without modification
type passThroughCodec struct{}

// Marshal takes a []byte pointer and passes it through as a []byte
func (passThroughCodec) Marshal(v interface{}) ([]byte, error) {
	bs, ok := v.(*[]byte)
	if !ok {
		return nil, fmt.Errorf("expected sender of type *[]byte but got %T", v)
	}
	return *(bs), nil
}

// Unmarshal takes a byte slice and writes it to v
func (passThroughCodec) Unmarshal(data []byte, v interface{}) error {
	bs, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("expected receiver of type *[]byte but got %T", v)
	}
	*bs = data
	return nil
}

func (passThroughCodec) String() string {
	return "passthrough"
}
