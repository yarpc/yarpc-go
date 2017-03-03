package iopool

type buffer struct {
	b []byte
}

func newBuffer(len int) *buffer {
	return &buffer{
		b: make([]byte, len),
	}
}
