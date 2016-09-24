package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yarpc/yarpc-go/transport"
)

/*  TODO Find a way to test the handle function, currently we can't mock the grpc stream and these tests crash
func TestHandler_Handle(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	decodedMsg := []byte("test string!")
	decoderFunc := func(v interface{}) error {
		bs, _ := v.(*[]byte)
		*bs = decodedMsg
		return nil
	}
	expectedBody := []byte("teeeeeest response")

	rpcHandler := transporttest.NewMockHandler(mockCtrl)
	grpcHandler := handler{Handler: rpcHandler}

	rpcHandler.EXPECT().Handle(
		ctx,
		grpcOptions,
		transporttest.NewRequestMatcher(
			t, &transport.Request{
				Caller:    "hello",
				Service:   "foo",
				Procedure: "bar",
				Encoding:  "raw",
				Body:      bytes.NewReader(decodedMsg),
			},
		),
		gomock.Any(),
	).Return(nil).Do(
		func(ctx context.Context, opts transport.Options, req *transport.Request, resw transport.ResponseWriter) {
			resw.Write(expectedBody)
		},
	)

	body, err := grpcHandler.Handle(nil, ctx, decoderFunc, nil)

	assert.Equal(t, error(nil), err)
	assert.Equal(t, expectedBody, *(body.(*[]byte)))
}

func TestHandler_Handle_HandlerError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	decodedMsg := []byte("test string!")
	decoderFunc := func(v interface{}) error {
		bs, _ := v.(*[]byte)
		*bs = decodedMsg
		return nil
	}
	expectedErr := errors.New("teeeest error")

	rpcHandler := transporttest.NewMockHandler(mockCtrl)
	grpcHandler := handler{Handler: rpcHandler}

	rpcHandler.EXPECT().Handle(
		ctx,
		grpcOptions,
		transporttest.NewRequestMatcher(
			t, &transport.Request{
				Caller:    "hello",
				Service:   "foo",
				Procedure: "bar",
				Encoding:  "raw",
				Body:      bytes.NewReader(decodedMsg),
			},
		),
		gomock.Any(),
	).Return(expectedErr)

	_, err := grpcHandler.Handle(nil, ctx, decoderFunc, nil)

	assert.Equal(t, expectedErr, err)
}

func TestHandler_Handle_DecoderError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	decodingError := errors.New("DecodingError")
	decoderFunc := func(v interface{}) error {
		return decodingError
	}

	rpcHandler := transporttest.NewMockHandler(mockCtrl)
	grpcHandler := handler{Handler: rpcHandler}

	res, err := grpcHandler.Handle(nil, ctx, decoderFunc, nil)

	assert.Equal(t, nil, res)
	assert.Equal(t, decodingError, err)
}
*/

func TestResponseWriter_Write(t *testing.T) {
	strMsg := "this is a test"
	byteMsg := []byte(strMsg)
	var r response
	rw := newResponseWriter(&r)

	changed, err := rw.Write(byteMsg)

	assert.Equal(t, len(byteMsg), changed)
	assert.Equal(t, error(nil), err)
	assert.Equal(t, strMsg, string(r.body.Bytes()))
}

func TestResponseWriter_AddHeaders(t *testing.T) {
	headers := transport.NewHeadersWithCapacity(10)
	var r response
	rw := newResponseWriter(&r)

	rw.AddHeaders(headers)

	assert.Equal(t, headers, r.headers)
}

func TestResponseWriter_SetApplicationError(t *testing.T) {
	var r response
	rw := newResponseWriter(&r)

	rw.SetApplicationError()

	// No action on Application Error
}
