package grpc

import "fmt"

// PassThroughCodec passes bytes to/from the wire without modification
type PassThroughCodec struct{}

// Marshal takes a []byte pointer and passes it through as a []byte
func (PassThroughCodec) Marshal(v interface{}) ([]byte, error) {
	bs, ok := v.(*[]byte)
	if !ok {
		return nil, fmt.Errorf("expected sender of type *[]byte but got %T", v)
	}
	return *(bs), nil
}

// Unmarshal takes a byte slice and writes it to v
func (PassThroughCodec) Unmarshal(data []byte, v interface{}) error {
	bs, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("expected receiver of type *[]byte but got %T", v)
	}
	*bs = data
	return nil
}

func (PassThroughCodec) String() string {
	return "raw"
}
