// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package debug

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"

	"go.uber.org/yarpc"

	"github.com/stretchr/testify/require"
	yarpchttp "go.uber.org/yarpc/transport/http"
)

var (
	_jsonTestTmpl = template.Must(template.New("tmpl").Funcs(template.FuncMap{
		"jsonMarshal": func(v interface{}) (string, error) {
			data, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	}).Parse(`{{jsonMarshal .}}`))

	_errorTestTmpl = template.Must(template.New("tmpl").Funcs(template.FuncMap{
		"returnError": func(_ interface{}) (string, error) {
			return "", errors.New("error")
		},
	}).Parse(`{{returnError .}}`))
)

func TestHandler(t *testing.T) {
	dispatcher := newTestDispatcher()

	expectedData, err := json.Marshal(newTmplData(dispatcher.Introspect()))
	require.NoError(t, err)

	responseRecorder := httptest.NewRecorder()
	NewHandler(dispatcher, tmpl(_jsonTestTmpl))(responseRecorder, nil)

	require.Equal(t, http.StatusOK, responseRecorder.Code)
	data, err := ioutil.ReadAll(responseRecorder.Body)
	require.NoError(t, err)
	require.Equal(t, string(expectedData), string(data))
}

func TestHandlerError(t *testing.T) {
	dispatcher := newTestDispatcher()

	responseRecorder := httptest.NewRecorder()
	NewHandler(dispatcher, tmpl(_errorTestTmpl))(responseRecorder, nil)
	require.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
}

func newTestDispatcher() *yarpc.Dispatcher {
	httpTransport := yarpchttp.NewTransport()
	return yarpc.NewDispatcher(yarpc.Config{
		Name: "test",
		Inbounds: yarpc.Inbounds{
			httpTransport.NewInbound("127.0.0.1:0"),
		},
		Outbounds: yarpc.Outbounds{
			"test-client": {
				Unary:  httpTransport.NewSingleOutbound("http://127.0.0.1:1234"),
				Oneway: httpTransport.NewSingleOutbound("http://127.0.0.1:1234"),
			},
		},
	})
}
