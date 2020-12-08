package protobuf

import (
	"fmt"
	"github.com/golang/protobuf/proto"
)

func gogoGhProtoMarshal(pb proto.Message,p *proto.Buffer) error {
	if m, ok := pb.(newMarshaler); ok {
		siz := m.XXX_Size()
		growBuffer(siz,p) // make sure buf has enough capacity
		pp := p.Bytes()[len(p.Bytes()) : len(p.Bytes()) : len(p.Bytes())+siz]
		pp, err := m.XXX_Marshal(pp, false)
		p.SetBuf(append(p.Bytes(),pp...))
		return err
	}
	if m, ok := pb.(proto.Marshaler); ok {
		b, err := m.Marshal()
		p.SetBuf(append(p.Bytes(),b...))
		return err
	}
	if pb == nil {
		return fmt.Errorf("no wrapper found for message")
	}
	var info proto.InternalMessageInfo
	siz := info.Size(pb)
	growBuffer(siz,p) // make sure buf has enough capacity
	byt, err := info.Marshal(p.Bytes(), pb, false)
	p.SetBuf(append(p.Bytes(),byt...))
	return err
}

func growBuffer(n int, p *proto.Buffer) {
	need := len(p.Bytes()) + n
	if need <= cap(p.Bytes()) {
		return
	}
	newCap := len(p.Bytes()) * 2
	if newCap < need {
		newCap = need
	}
	p.SetBuf(append(make([]byte, 0, newCap), p.Bytes()...))
}


type newMarshaler interface {
	XXX_Size() int
	XXX_Marshal(b []byte, deterministic bool) ([]byte, error)
}
