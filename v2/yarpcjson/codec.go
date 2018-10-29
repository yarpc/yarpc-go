package yarpcjson

import (
	"encoding/json"
	"reflect"

	"go.uber.org/yarpc/v2"
)

type jsonCodec struct {
	reader requestReader
}

func newCodec(name string, handler interface{}) jsonCodec {
	reqBodyType := verifyUnarySignature(name, reflect.TypeOf(handler))
	var r requestReader
	if reqBodyType == _interfaceEmptyType {
		r = ifaceEmptyReader{}
	} else if reqBodyType.Kind() == reflect.Map {
		r = mapReader{reqBodyType}
	} else {
		// struct ptr
		r = structReader{reqBodyType.Elem()}
	}

	return jsonCodec{
		reader: r,
	}
}

func (c jsonCodec) Decode(res *yarpc.Buffer) (interface{}, error) {
	reqBody, err := c.reader.Read(json.NewDecoder(res))
	if err != nil {
		return nil, err
	}

	return reqBody.Interface(), nil
}

func (c jsonCodec) Encode(res interface{}) (*yarpc.Buffer, error) {
	resBuf := &yarpc.Buffer{}
	if err := json.NewEncoder(resBuf).Encode(res); err != nil {
		return nil, err
	}

	return resBuf, nil
}
