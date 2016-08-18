package grpc

// RawCodec writes strings to/from the wire
type RawCodec struct{}

// Marshal takes a string pointer converts it to a byte sliced
func (RawCodec) Marshal(v interface{}) ([]byte, error) {
	return []byte(*(v.(*string))), nil
}

// Unmarshal takes a byte slice and writes it to v
func (RawCodec) Unmarshal(data []byte, v interface{}) error {
	*(v.(*string)) = string(data)
	return nil
}

func (RawCodec) String() string {
	return "raw"
}
