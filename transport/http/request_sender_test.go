package http

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSender(t *testing.T) {
	const data = "dummy server response body"

	var (
		server = httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				io.WriteString(w, data)
			},
		))
		clientReq, _ = http.NewRequest("GET", server.URL, nil)
		serverReq    = httptest.NewRequest("GET", server.URL, nil)
		client       = &http.Client{
			Transport: http.DefaultTransport,
		}
	)
	defer server.Close()

	tests := []struct {
		msg            string
		sender         sender
		req            *http.Request
		wantStatusCode int
		wantBody       string
		wantError      string
	}{
		{
			msg:            "http.Client sender, http client request",
			req:            clientReq,
			sender:         http.DefaultClient,
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
		{
			msg:       "http.Client sender, http server request",
			req:       serverReq,
			sender:    http.DefaultClient,
			wantError: "http: Request.RequestURI can't be set in client requests.",
		},
		{
			msg:            "transportSender, http client request",
			req:            clientReq,
			sender:         &transportSender{Client: client},
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
		{
			msg:            "transportSender, http server request",
			req:            serverReq,
			sender:         &transportSender{Client: client},
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			resp, err := tt.sender.Do(tt.req)
			if tt.wantError != "" {
				require.Error(t, err, "expect error when we use http.Client to send a server request")
				assert.Contains(t, err.Error(), tt.wantError, "error body mismatch")
				return
			}
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode, "status code does not match")
			body, _ := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantBody, string(body), "response body does not match")
		})
	}
}
