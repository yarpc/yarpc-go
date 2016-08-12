package grpc

type rawCodec struct {
}

func (rawCodec) Marshal(v interface{}) ([]byte, error) {
	return []byte(*(v.(*string))), nil
}

func (rawCodec) Unmarshal(data []byte, v interface{}) error {
	*(v.(*string)) = string(data)
	return nil
}

func (rawCodec) String() string {
	return "raw"
}
